/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package usage

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/usagerecord"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	CPUCost    = 0.000003
	MemoryCost = 0.000004
)

var (
	usage = new(Usage)
	mutex = new(sync.Mutex)
)

type UUID string
type Limit struct {
	Time   int64
	CPU    resource.Quantity
	Memory resource.Quantity
}

type RawResource struct {
	LastReportTime int64
	Limit          []Limit
}

type Resource struct {
	// UUID is resource uid
	Resource           map[UUID]RawResource
	SubscriptionItemID string
}

type GroupResource struct {
	GroupResource map[string]Resource
}

type Usage struct {
	// UUID is cluster uid
	Cluster map[UUID]GroupResource
}

func ReportUsage() {
	ticker := time.NewTicker(time.Minute * 2)
	for {
		select {
		case now := <-ticker.C:
			log.Printf("Starting to report usage at %v\n", now)
			mutex.Lock()

			for cid, cluster := range usage.Cluster {
				for gr, group := range cluster.GroupResource {
					sii := group.SubscriptionItemID
					for rid, rs := range group.Resource {
						q := calculateTotalQuantity(now.Unix(), rs.LastReportTime, rs.Limit)
						if q == 0 {
							continue
						}
						if err := recordUsage(sii, q); err != nil {
							return
						}
						rs.LastReportTime = now.Unix()
						group.Resource[rid] = rs

						log.Printf("Usage reported q: %v for %s at %v\n", q, sii, now)
					}
					cluster.GroupResource[gr] = group
				}
				usage.Cluster[cid] = cluster
			}
			log.Println("Reporting usage done for now...")
			mutex.Unlock()
		}
	}
}

func calculateTotalQuantity(now, last int64, q []Limit) int64 {
	sort.Slice(q, func(i, j int) bool { return q[i].Time > q[j].Time })
	idx := sort.Search(len(q), func(i int) bool { return q[i].Time >= last })

	if idx >= len(q) {
		return 0
	}
	sum := Limit{}
	pre := Limit{Time: last}
	if q[idx].Time > last && idx > 0 {
		pre.CPU = q[idx-1].CPU
		pre.Memory = q[idx-1].Memory
	}

	tmp := resource.MustParse("0")
	for _, l := range q {
		tmp.SetMilli(pre.CPU.MilliValue() * (l.Time - pre.Time))
		sum.CPU.Add(tmp)

		tmp.SetMilli(pre.Memory.MilliValue() * (l.Time - pre.Time))
		sum.Memory.Add(tmp)

		pre = Limit{
			Time:   l.Time,
			CPU:    l.CPU,
			Memory: l.Memory,
		}
	}

	if now > pre.Time {
		tmp.SetMilli(pre.CPU.MilliValue() * (now - pre.Time))
		sum.CPU.Add(tmp)

		tmp.SetMilli(pre.Memory.MilliValue() * (now - pre.Time))
		sum.Memory.Add(tmp)
	}

	return int64(float64(sum.CPU.MilliValue())*CPUCost + float64(sum.Memory.MilliValue())*MemoryCost)
}

func recordUsage(sii string, q int64) error {
	params := &stripe.UsageRecordParams{
		Action:           stripe.String(stripe.UsageRecordActionIncrement),
		Quantity:         stripe.Int64(q),
		SubscriptionItem: stripe.String(sii),
		Timestamp:        stripe.Int64(time.Now().Unix()),
	}

	_, err := usagerecord.New(params)
	if err != nil {
		return err
	}

	return nil
}

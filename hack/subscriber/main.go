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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/usagerecord"

	cnats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/the-redback/go-oneliners"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/core"
	"kmodules.xyz/resource-metadata/hub"
)

const (
	CPUCost    = 0.000003
	MemoryCost = 0.000004
)

var (
	classicUrl = "nats://localhost:4222"
	usage      = new(Usage)
	mutex      = new(sync.Mutex)
)

func init() {
	stripe.Key = "sk_test_3zxmVmMJBuVYwNnBeKwB7msF"
}

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

func main() {
	if len(os.Args) > 1 {
		classicUrl = os.Args[1]
	}
	con, err := cnats.NewConsumer(classicUrl, "ClusterEvents", cnats.NatsOptions())
	if err != nil {
		log.Fatal(err)
	}
	defer con.Close(context.Background())
	log.Printf("Connected to nats server: %s\n", classicUrl)

	client, err := cloudevents.NewClient(con)
	if err != nil {
		panic(err)
	}

	go ReportUsage()

	for {
		err := client.StartReceiver(context.Background(), processEvents)
		if err != nil {
			log.Printf("failed to start nats receiver, #{%s}", err.Error())
		}
	}
}

func processEvents(ctx context.Context, event cloudevents.Event) error {
	eventType := strings.Split(event.Type(), "$")
	var data = make(map[string]interface{})
	if err := event.DataAs(&data); err != nil {
		log.Println(err)
		return err
	}
	u := new(unstructured.Unstructured)
	if err := u.UnmarshalJSON(event.Data()); err != nil {
		log.Println(err)
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    eventType[1],
		Version:  eventType[2],
		Resource: eventType[3],
	}
	reg := hub.NewRegistryOfKnownResources()
	rd, err := reg.LoadByGVR(gvr)
	if err != nil {
		return err
	}

	requirements := &core.ResourceRequirements{
		Limits:   core.ResourceList{},
		Requests: core.ResourceList{},
	}
	cpuSum := requirements.Limits[core.ResourceCPU]
	memorySum := requirements.Limits[core.ResourceMemory]

	for _, re := range rd.Spec.ResourceRequirements {
		// Extract resource requests & limits from yaml
		replicas, ok, err := unstructured.NestedInt64(u.Object, strings.Split(re.Units, ".")...)
		if err = handleUnstructuredError(err, ok, re.Units); err != nil {
			log.Println(err)
			continue
		}
		resources, ok, err := unstructured.NestedMap(u.Object, strings.Split(re.Resources, ".")...)
		if err = handleUnstructuredError(err, ok, re.Units); err != nil {
			log.Println(err)
			continue
		}
		var shards = int64(1)
		if len(re.Shards) > 0 {
			shards, ok, err = unstructured.NestedInt64(u.Object, strings.Split(re.Shards, ".")...)
			if err = handleUnstructuredError(err, ok, re.Units); err != nil {
				log.Println(err)
				continue
			}
		}

		units := replicas * shards
		tmp := new(core.ResourceRequirements)

		if err := UnmarshalMapToStruct(resources, tmp); err != nil {
			log.Println(err)
		}

		// If limits doesn't exist
		// take requests as cpu & memory resources
		cpu := tmp.Limits[core.ResourceCPU]
		if tmp.Limits.CPU() == nil {
			cpu = tmp.Requests[core.ResourceCPU]
		}
		memory := tmp.Limits[core.ResourceMemory]
		if tmp.Limits.Memory() == nil {
			memory = tmp.Requests[core.ResourceMemory]
		}

		cpu.SetMilli(cpu.MilliValue() * units)
		memory.SetMilli(memory.MilliValue() * units)

		cpuSum.Add(cpu)
		memorySum.Add(memory)
	}

	requirements.Limits[core.ResourceCPU] = cpuSum
	requirements.Limits[core.ResourceMemory] = memorySum

	oneliners.PrettyJson(requirements.Limits, fmt.Sprintf("%s ns=%s, name=%s, op=%s", gvr.String(), eventType[4], eventType[5], eventType[6]))

	mutex.Lock()
	defer mutex.Unlock()

	cid := UUID(eventType[0])
	gr := gvr.GroupResource().String()
	rid := UUID(u.GetUID())
	ensureInitialUsage(cid, rid, gr)
	usage.Cluster[cid].GroupResource[gr] = Resource{
		Resource: usage.Cluster[cid].GroupResource[gr].Resource,
		// TODO: Fetch subscription id from source
		// SubscriptionItemID: eventType[0],
	}

	raw := usage.Cluster[cid].GroupResource[gr].Resource[rid]
	switch eventType[len(eventType)-1] {
	case "delete":
		deletionTime := event.Time().Unix()
		if u.GetDeletionTimestamp() != nil {
			deletionTime = u.GetDeletionTimestamp().Unix()
		}
		if len(raw.Limit) == 0 {
			raw.Limit = append(raw.Limit, Limit{
				Time:   u.GetCreationTimestamp().Unix(),
				CPU:    cpuSum,
				Memory: memorySum,
			},
			)
		}
		raw.Limit = append(raw.Limit, Limit{
			Time:   deletionTime,
			CPU:    resource.Quantity{},
			Memory: resource.Quantity{},
		},
		)

	case "update":
		raw.Limit = append(raw.Limit, Limit{
			Time:   event.Time().Unix(),
			CPU:    cpuSum,
			Memory: memorySum,
		},
		)
	case "create":
		raw.Limit = append(raw.Limit, Limit{
			Time:   u.GetCreationTimestamp().Unix(),
			CPU:    cpuSum,
			Memory: memorySum,
		},
		)
	}
	usage.Cluster[cid].GroupResource[gr].Resource[rid] = raw

	//now := time.Now()
	//q := calculateTotalQuantity(now.Unix(), raw.LastReportTime, raw.Limit)
	//if err := recordUsage(usage.Cluster[cid].GroupResource[gr].SubscriptionItemID, q); err != nil {
	//	return err
	//}
	//raw.LastReportTime = now.Unix()
	//usage.Cluster[cid].GroupResource[gr].Resource[rid] = raw

	oneliners.PrettyJson(usage, "Total Usage Report")
	//log.Printf("Usage reported q: %v for %s at %v\n", q, usage.Cluster[cid].GroupResource[gr].SubscriptionItemID, now)

	return nil
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

func ensureInitialUsage(cid, rid UUID, gr string) {
	if usage.Cluster == nil {
		usage.Cluster = map[UUID]GroupResource{}
	}
	cluster, ok := usage.Cluster[cid]
	if !ok {
		cluster = GroupResource{GroupResource: map[string]Resource{}}
	}

	groupR, ok := cluster.GroupResource[gr]
	if !ok {
		groupR = Resource{Resource: map[UUID]RawResource{}}
	}

	res, ok := groupR.Resource[rid]
	if !ok {
		res = RawResource{LastReportTime: 0, Limit: make([]Limit, 0)}
	}
	groupR.Resource[rid] = res
	cluster.GroupResource[gr] = groupR
	usage.Cluster[cid] = cluster

	return
}

func handleUnstructuredError(err error, ok bool, field string) error {
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("field `%s` doesn't exist", field)
	}
	return nil
}

//UnmarshalMapToStruct
func UnmarshalMapToStruct(mapData map[string]interface{}, structObj interface{}) error {
	data, err := json.Marshal(mapData)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, structObj); err != nil {
		return err
	}
	return nil
}

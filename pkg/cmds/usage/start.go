package usage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	cnats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"github.com/the-redback/go-oneliners"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/core"
	"kmodules.xyz/resource-metadata/hub"
)

type NatsOptions struct {
	Server         string
	CredentialFile string
	Subject        string
}

func NewNatsOptions() *NatsOptions {
	return &NatsOptions{
		Server:  "nats://localhost:4222",
		Subject: "ClusterEvents",
	}
}
func (opts *NatsOptions) AddGoFlags(fs *flag.FlagSet) {
	fs.StringVar(&opts.Server, "server", opts.Server, "Nats server endpoint")
	fs.StringVar(&opts.CredentialFile, "credential-file", opts.CredentialFile, "User credential for connecting to nats server")
	fs.StringVar(&opts.Subject, "subject", opts.Subject, "Channel name from which events to be listened")
}

func (opts *NatsOptions) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("usage", flag.ExitOnError)
	opts.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (opts *NatsOptions) StartNatsSubscription() {
	fmt.Println("Server", "<==>", opts.Server)
	fmt.Println("Subject", "<==>", opts.Subject)
	var natsOpts = []nats.Option{nats.Name("Usage Report")}
	if len(opts.CredentialFile) > 0 {
		natsOpts = append(natsOpts, nats.UserCredentials(opts.CredentialFile))
	}

	con, err := cnats.NewConsumer(opts.Server, opts.Subject, cnats.NatsOptions(natsOpts...))
	if err != nil {
		log.Fatal(err)
	}
	defer con.Close(context.Background())
	log.Printf("Connected to nats server: %s\n", opts.Server)

	client, err := cloudevents.NewClient(con)
	if err != nil {
		panic(err)
	}

	go ReportUsage()

	for {
		println("starting subscriber")
		err := client.StartReceiver(context.Background(), processEvents)
		if err != nil {
			log.Printf("failed to start nats receiver, #{%s}", err.Error())
		}
	}
}

func processEvents(ctx context.Context, event cloudevents.Event) error {
	println("hello")
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

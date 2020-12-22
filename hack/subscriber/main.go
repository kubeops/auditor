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
	"strconv"
	"strings"

	cnats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/the-redback/go-oneliners"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/core"
	"kmodules.xyz/resource-metadata/hub"
)

var (
	classicUrl string = "nats://localhost:4222"
)

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
		replicas, ok, err := unstructured.NestedFloat64(data, strings.Split(re.Units, ".")...)
		if err = handleUnstructuredError(err, ok, re.Units); err != nil {
			log.Println(err)
			continue
		}
		resources, ok, err := unstructured.NestedMap(data, strings.Split(re.Resources, ".")...)
		if err = handleUnstructuredError(err, ok, re.Units); err != nil {
			log.Println(err)
			continue
		}
		var shards = float64(1)
		if len(re.Shards) > 0 {
			shards, ok, err = unstructured.NestedFloat64(data, strings.Split(re.Shards, ".")...)
			if err = handleUnstructuredError(err, ok, re.Units); err != nil {
				log.Println(err)
				continue
			}
		}

		units := int64(replicas * shards)
		tmp := new(core.ResourceRequirements)

		if err := UnmarshalMapToStruct(resources, tmp); err != nil {
			log.Println(err)
		}
		cpu := tmp.Limits[core.ResourceCPU]
		memory := tmp.Limits[core.ResourceMemory]

		number, suffix := CanonicalizeResourceQuantity(cpu)
		cpu, err = resource.ParseQuantity(fmt.Sprintf("%v%s", number*units, suffix))
		if err != nil {
			log.Println(err)
		}

		number, suffix = CanonicalizeResourceQuantity(memory)
		memory, err = resource.ParseQuantity(fmt.Sprintf("%v%s", number*units, suffix))
		if err != nil {
			log.Println(err)
		}

		cpuSum.Add(cpu)
		memorySum.Add(memory)
	}

	requirements.Limits[core.ResourceCPU] = cpuSum
	requirements.Limits[core.ResourceMemory] = memorySum

	oneliners.PrettyJson(requirements.Limits, gvr.String())

	return nil
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

func CanonicalizeResourceQuantity(q resource.Quantity) (number int64, suffix string) {
	result := make([]byte, 0, 18)
	num, sfx := q.CanonicalizeBytes(result)
	number, _ = strconv.ParseInt(string(num), 10, 0)
	suffix = string(sfx)
	return
}

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

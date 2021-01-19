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
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/golangplus/testing/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubernetes/pkg/apis/core"
	"sigs.k8s.io/yaml"
)

func TestUnstructuredNestedMap(t *testing.T) {
	data, err := ioutil.ReadFile("es.yaml")
	assert.NoError(t, err)
	obj := make(map[string]interface{})
	err = yaml.Unmarshal(data, &obj)
	assert.NoError(t, err)

	replicas, ok, err := unstructured.NestedFloat64(obj, strings.Split("spec.topology.master.replicas", ".")...)
	assert.NoError(t, err)
	assert.True(t, "`units`", ok)
	resources, ok, err := unstructured.NestedMap(obj, strings.Split("spec.topology.master.resources", ".")...)
	assert.NoError(t, err)
	assert.True(t, "`resources`", ok)
	shards, ok, err := unstructured.NestedFloat64(obj, strings.Split("spec.topology.master.shards", ".")...)
	assert.NoError(t, err)
	if !ok {
		shards = 1
	}

	units := int64(replicas * shards)

	rse := new(core.ResourceRequirements)
	err = UnmarshalMapToStruct(resources, rse)
	assert.NoError(t, err)

	cpu := rse.Limits[core.ResourceCPU]
	cpu.SetMilli(cpu.MilliValue() * units)
	rse.Limits[core.ResourceCPU] = cpu
	fmt.Println(rse.Limits.CPU().String())
}

func TestQuantity(t *testing.T) {
	cpu, err := resource.ParseQuantity("0")
	assert.NoError(t, err)
	cpu.SetMilli(100)
	fmt.Println(cpu.String())
}

func TestUsage(t *testing.T) {
	cid, rid, gr := UUID("cid"), UUID("rid"), "elasticsearches.kubedb.com"
	if usage.Cluster == nil {
		usage.Cluster = map[UUID]GroupResource{}
	}
	cluster, ok := usage.Cluster[cid]
	if !ok {
		cluster = GroupResource{
			GroupResource: map[string]Resource{},
		}
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

	// testing delete operation
	cpuSum := resource.Quantity{}
	memorySum := resource.Quantity{}
	var err error
	cpuSum, err = resource.ParseQuantity("200m")
	assert.NoError(t, err)
	memorySum, err = resource.ParseQuantity("100Mi")
	assert.NoError(t, err)

	fmt.Println(cpuSum.String(), "<==>", cpuSum.MilliValue())
	fmt.Println(memorySum.String(), "<==>", memorySum.MilliValue())

	raw := usage.Cluster[cid].GroupResource[gr].Resource[rid]
	if len(usage.Cluster[cid].GroupResource[gr].Resource[rid].Limit) == 0 {
		raw.Limit = append(raw.Limit, Limit{
			Time:   time.Now().Unix(),
			CPU:    cpuSum,
			Memory: memorySum,
		},
		)
		//oneliners.PrettyJson(usage.Cluster[cid].GroupResource[gr].Resource[rid], "Before")
	}
	raw.Limit = append(raw.Limit, Limit{
		Time:   time.Now().Unix(),
		CPU:    resource.Quantity{},
		Memory: resource.Quantity{},
	},
	)

	usage.Cluster[cid].GroupResource[gr].Resource[rid] = raw
	//oneliners.PrettyJson(usage.Cluster[cid].GroupResource[gr].Resource[rid], "After")
}

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

	number, suffix := CanonicalizeResourceQuantity(*rse.Limits.CPU())

	rse.Limits[core.ResourceCPU] = resource.MustParse(fmt.Sprintf("%v%s", number*units, suffix))
	fmt.Println(rse.Limits.CPU().String())
}

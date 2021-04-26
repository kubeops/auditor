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
	"path/filepath"
	"runtime"

	"kubeops.dev/auditor/apis/auditor/v1alpha1"

	"gomodules.xyz/x/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var (
	_, filePath, _, _ = runtime.Caller(0)
	policyDirectory   = filepath.Dir(filePath)
)

func main() {
	policy := &v1alpha1.AuditRegistration{
		TypeMeta: v1.TypeMeta{
			Kind:       v1alpha1.ResourceKindAuditRegistration,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Resources: []v1alpha1.GroupResources{
			{
				Group:     "apps",
				Resources: []string{"deployments"},
			},
			{
				Resources: []string{"pods", "namespaces", "secrets"},
			},
			{
				Group:     "appcatalog.appscode.com",
				Resources: []string{"appbindings"},
			},
			{
				Group:     "catalog.kubedb.com",
				Resources: []string{"etcdversions", "mysqlversions", "redisversions", "mongodbversions", "postgresversions", "memcachedversions", "elasticsearchversions"},
			},
			{
				Group:     "cloud.bytebuilders.dev",
				Resources: []string{"credentials", "machinetypes", "cloudproviders", "clusterinfos", "clusteruserauths", "clusterauthinfotemplates"},
			},
			{
				Group:     "kubedb.com",
				Resources: []string{"etcds", "mysqls", "redises", "mongodbs", "snapshots", "memcacheds", "postgreses", "elasticsearches", "dormantdatabases"},
			},
			{
				Group:     "kubepack.com",
				Resources: []string{"plans", "products"},
			},
			{
				Group:     "monitoring.appscode.com",
				Resources: []string{"incidents", "podalerts", "nodealerts", "clusteralerts", "searchlightplugins"},
			},
			{
				Group:     "stash.appscode.com",
				Resources: []string{"tasks", "restics", "functions", "recoveries", "repositories", "backupbatches", "backupsessions", "restoresessions", "backupblueprints", "backupconfigurations"},
			},
			{
				Group:     "voyager.appscode.com",
				Resources: []string{"ingresses", "certificates"},
			},
		},
	}

	data, err := yaml.Marshal(policy)
	if err != nil {
		panic(err)
	}

	ioutil.WriteString(filepath.Join(policyDirectory, "default-policy.yaml"), string(data))
}

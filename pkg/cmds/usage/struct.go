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

import "fmt"

type LocalInvoiceList struct {
	Items []LocalInvoice
}

type LocalInvoice struct {
	Cluster       string
	Product       string
	Instance      ResourceInstance
	CurrentState  UsageStatus
	PreviousState UsageStatus
}

type ResourceInstance struct {
	Namespace string
	Name      string
	UID       string
}

type UsageStatus struct {
	CPU    string
	Memory string
	Time   int64
}

func (ri ResourceInstance) String() string {
	return fmt.Sprintf("%s/%s, %s", ri.Namespace, ri.Name, ri.UID)
}

func (ru UsageStatus) String() string {
	return fmt.Sprintf("%s cpu, %s memory", ru.CPU, ru.Memory)
}

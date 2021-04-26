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

package framework

import (
	"gomodules.xyz/x/crypto/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Framework struct {
	restConfig *rest.Config
	kubeClient kubernetes.Interface

	namespace string
	name      string
}

func New(
	restConfig *rest.Config,
	kubeClient kubernetes.Interface,
) *Framework {
	return &Framework{
		restConfig: restConfig,
		kubeClient: kubeClient,

		name:      rand.WithUniqSuffix("auditor"),
		namespace: rand.WithUniqSuffix("auditor"),
	}
}

func (f *Framework) Invoke() *Invocation {
	return &Invocation{
		Framework: f,
		app:       rand.WithUniqSuffix("auditor-e2e"),
	}
}

func (f *Framework) Name() string {
	return f.name
}

func (fi *Invocation) KubeClient() kubernetes.Interface {
	return fi.kubeClient
}

func (fi *Invocation) RestConfig() *rest.Config {
	return fi.restConfig
}

type Invocation struct {
	*Framework
	app string
}

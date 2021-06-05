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

package controller

import (
	"time"

	"kubeops.dev/auditor/pkg/eventer"

	"go.bytebuilders.dev/audit/lib"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/custom-resources/apis/auditor/v1alpha1"
)

type config struct {
	LicenseFile string

	Policy v1alpha1.AuditRegistration

	MaxNumRequeues int
	NumThreads     int
	ResyncPeriod   time.Duration
}

type Config struct {
	config

	ClientConfig  *rest.Config
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
}

func NewConfig(clientConfig *rest.Config) *Config {
	return &Config{
		ClientConfig: clientConfig,
	}
}

func (c *Config) New() (*AuditorController, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	natscfg, err := lib.NewNatsConfig(c.KubeClient.CoreV1().Namespaces(), c.LicenseFile)
	if err != nil {
		return nil, err
	}

	ctrl := &AuditorController{
		config:                 c.config,
		clientConfig:           c.ClientConfig,
		kubeClient:             c.KubeClient,
		dynamicClient:          c.DynamicClient,
		dynamicInformerFactory: dynamicinformer.NewDynamicSharedInformerFactory(c.DynamicClient, c.ResyncPeriod),
		recorder:               eventer.NewEventRecorder(c.KubeClient, "auditor"),

		nats: natscfg,
	}
	return ctrl, nil
}

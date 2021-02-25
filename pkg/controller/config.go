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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"kmodules.xyz/auditor/apis/auditor/v1alpha1"
	"kmodules.xyz/auditor/pkg/eventer"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/clusterid"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
	verifier "go.bytebuilders.dev/license-verifier"
	"go.bytebuilders.dev/license-verifier/info"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type config struct {
	LicenseFile string

	Policy v1alpha1.AuditRegistration

	// TODO: Should include full HTTP endpoint options
	ReceiverAddress        string
	ReceiverCredentialFile string

	MaxNumRequeues int
	NumThreads     int
	ResyncPeriod   time.Duration
}

type Config struct {
	config

	ClientConfig  *rest.Config
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface

	NatsClient        *nats.Conn
	CloudEventsClient cloudevents.Client
}

func NewConfig(clientConfig *rest.Config) *Config {
	return &Config{
		ClientConfig: clientConfig,
	}
}

// the api response of the register licensed user api
type NatsCredential struct {
	NatsSubject string `json:"natsSubject"`
	NatsServer  string `json:"natsServer"`
	Credential  []byte `json:"credential"`
}

func (c *Config) New() (*AuditorController, error) {
	if err := discovery.IsDefaultSupportedVersion(c.KubeClient); err != nil {
		return nil, err
	}

	clusteruid, err := clusterid.ClusterUID(c.KubeClient.CoreV1().Namespaces())
	if err != nil {
		return nil, err
	}
	licenseBytes, err := ioutil.ReadFile(c.LicenseFile)
	if err != nil {
		return nil, err
	}

	opts := verifier.Options{
		ClusterUID:  clusteruid,
		ProductName: info.ProductName,
		CACert:      []byte(info.LicenseCA),
		License:     licenseBytes,
	}
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}

	// TODO: Make URL dynamic
	url := "https://appscode.ninja/api/v1/register"
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, err
	}
	var natscred NatsCredential
	err = json.Unmarshal(buf.Bytes(), &natscred)
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

		natsClient:        c.NatsClient,
		cloudEventsClient: c.CloudEventsClient,

		NatsCredential: natscred,
	}

	if err := ctrl.initWatchers(); err != nil {
		return nil, err
	}
	return ctrl, nil
}

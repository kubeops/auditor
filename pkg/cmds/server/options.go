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

package server

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"kubeshield.dev/auditor/apis/auditor/v1alpha1"
	"kubeshield.dev/auditor/pkg/controller"
	"kubeshield.dev/auditor/pkg/controller/receiver"

	natsevents "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
	"github.com/spf13/pflag"
	"gomodules.xyz/x/log"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"kmodules.xyz/client-go/tools/clusterid"
	"sigs.k8s.io/yaml"
)

type ExtraOptions struct {
	PolicyFile string

	// TODO: Should include full HTTP endpoint options, eg, CA, client certs
	// eg: https://github.com/DirectXMan12/k8s-prometheus-adapter/blob/master/cmd/adapter/adapter.go#L57-L66
	ReceiverAddress        string
	ReceiverCredentialFile string

	MaxNumRequeues int
	NumThreads     int
	QPS            float64
	Burst          int
	ResyncPeriod   time.Duration
}

func NewExtraOptions() *ExtraOptions {
	return &ExtraOptions{
		MaxNumRequeues: 5,
		NumThreads:     2,
		QPS:            100,
		Burst:          100,
		ResyncPeriod:   10 * time.Minute,
	}
}

func (s *ExtraOptions) AddGoFlags(fs *flag.FlagSet) {
	clusterid.AddGoFlags(fs)

	fs.StringVar(&s.PolicyFile, "policy-file", s.PolicyFile, "Path to policy file used to watch Kubernetes resources")

	fs.StringVar(&s.ReceiverAddress, "receiver-addr", "nats://classic-server.nats.svc", "Receiver endpoint address")
	fs.StringVar(&s.ReceiverCredentialFile, "receiver-credential-file", s.ReceiverCredentialFile, "Token used to authenticate with receiver")

	fs.Float64Var(&s.QPS, "qps", s.QPS, "The maximum QPS to the master from this client")
	fs.IntVar(&s.Burst, "burst", s.Burst, "The maximum burst for throttle")
	fs.DurationVar(&s.ResyncPeriod, "resync-period", s.ResyncPeriod, "If non-zero, will re-list this often. Otherwise, re-list will be delayed aslong as possible (until the upstream source closes the watch or times out.")
}

func (s *ExtraOptions) AddFlags(fs *pflag.FlagSet) {
	pfs := flag.NewFlagSet("auditor", flag.ExitOnError)
	s.AddGoFlags(pfs)
	fs.AddGoFlagSet(pfs)
}

func (s *ExtraOptions) ApplyTo(cfg *controller.Config) error {
	var err error

	if s.PolicyFile != "" {
		data, err := ioutil.ReadFile(s.PolicyFile)
		if err != nil {
			return fmt.Errorf("failed to read policy file: %v", err)
		}
		var policy v1alpha1.AuditRegistration
		err = yaml.Unmarshal(data, &policy)
		if err != nil {
			return fmt.Errorf("failed to parse policy file: %v", err)
		}
		cfg.Policy = policy
	}
	cfg.Policy.Resources = append(cfg.Policy.Resources, []v1alpha1.GroupResources{
		{
			Group:     "apps",
			Resources: []string{"deployments"},
		},
		{
			Resources: []string{"pods"},
		},
	}...)
	cfg.ReceiverAddress = s.ReceiverAddress
	cfg.ReceiverCredentialFile = s.ReceiverCredentialFile

	cfg.MaxNumRequeues = s.MaxNumRequeues
	cfg.NumThreads = s.NumThreads
	cfg.ResyncPeriod = s.ResyncPeriod
	cfg.ClientConfig.QPS = float32(s.QPS)
	cfg.ClientConfig.Burst = s.Burst

	if cfg.KubeClient, err = kubernetes.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	if cfg.DynamicClient, err = dynamic.NewForConfig(cfg.ClientConfig); err != nil {
		return err
	}
	var natsOpts = []nats.Option{nats.Name("Auditor")}
	if len(cfg.ReceiverCredentialFile) > 0 {
		natsOpts = append(natsOpts, nats.UserCredentials(cfg.ReceiverCredentialFile))
	}
	if cfg.NatsClient, err = nats.Connect(cfg.ReceiverAddress, natsOpts...); err != nil {
		return err
	}
	log.Infof("Nats connection established to %s", cfg.ReceiverAddress)

	sender, err := natsevents.NewSenderFromConn(cfg.NatsClient, receiver.Subject)
	if err != nil {
		return err
	}

	if cfg.CloudEventsClient, err = cloudevents.NewClient(sender); err != nil {
		return err
	}

	return nil
}

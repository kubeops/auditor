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
	"flag"
	"fmt"

	"github.com/nats-io/nats.go"
	api "go.bytebuilders.dev/audit/api/v1"
	"go.bytebuilders.dev/audit/lib"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kmapi "kmodules.xyz/client-go/api/v1"
	"sigs.k8s.io/yaml"
)

// You have to acknowledge the event from nats server
// you can do it by running `nats reply "subject"`

func main() {
	server := flag.String("server", "nats://localhost:4222", "NATS server url")
	subject := flag.String("subject", "subject", "Subject to which the event will be published")
	creds := flag.String("creds", "", "NATS credential path")

	fmt.Println("server: ", *server, "\nsubject: ", *subject, "\ncredential: ", *creds)

	nc, err := getNatsClient(*server, *creds)
	if err != nil {
		panic(err)
	}

	opEvent, err := getExampleReceiverEvent()
	if err != nil {
		panic(err)
	}

	publisher := lib.NewEventPublisher(&lib.NatsConfig{
		LicenseID: "xyz",
		Subject:   *subject,
		Server:    *server,
		Client:    nc,
	}, nil, nil)

	if err = publisher.Publish(opEvent, "example"); err != nil {
		panic(err)
	}
}

func getNatsClient(server, creds string) (*nats.Conn, error) {
	var natsOpts = []nats.Option{nats.Name("Auditor")}

	if len(creds) > 0 {
		natsOpts = append(natsOpts, nats.UserCredentials(creds))
	}
	nc, err := nats.Connect(server, natsOpts...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

func getExampleReceiverEvent() (*api.Event, error) {
	pod := `apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      image: nginx
      ports:
        - name: web
          containerPort: 80
          protocol: TCP`

	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(pod), u); err != nil {
		return nil, err
	}

	return &api.Event{
		Resource: u,
		ResourceID: kmapi.ResourceID{
			Group:   "",
			Version: "v1",
			Name:    "pods",
			Kind:    "Pod",
			Scope:   "Namespaced",
		},
		LicenseID: "abc-xyz-123",
	}, nil
}

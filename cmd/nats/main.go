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
	"os"

	"kubeops.dev/auditor/pkg/controller/receiver"

	"github.com/nats-io/nats.go"
)

func main() {
	if len(os.Args) < 4 {
		panic("argument missing")
	}

	server := os.Args[1]
	subject := os.Args[2]
	creds := os.Args[3]

	fmt.Println(server, subject, creds)

	nc, err := getNatsClient(server, creds)
	if err != nil {
		panic(err)
	}

	if err = receiver.PublishEvent(nc, subject, "test", "Hello there"); err != nil {
		panic(err)
	}

	//if _, err = nc.Request(subject, []byte("Hello there"), time.Second*5); err != nil {
	//	panic(err)
	//}
}

func getNatsClient(server, creds string) (*nats.Conn, error) {
	var natsOpts = []nats.Option{nats.Name("Auditor")}

	natsOpts = append(natsOpts, nats.UserCredentials(creds))
	nc, err := nats.Connect(server, natsOpts...)
	if err != nil {
		return nil, err
	}

	return nc, nil
}

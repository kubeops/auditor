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

	cnats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
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

	_, nc, err := cloudEventsClient(server, subject, creds)
	if err != nil {
		panic(err)
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		fmt.Println(string(msg.Data))
		_ = msg.Respond([]byte(fmt.Sprintf("message received from %s", msg.Subject)))
	})
	if err != nil {
		panic(err)
	}
	defer sub.Unsubscribe()

	select {}

	//err = receiver.PublishEvent(client, subject, "test", []byte("Hello"))
	//if err != nil {
	//	panic(err)
	//}
}

func cloudEventsClient(server, subject, creds string) (cloudevents.Client, *nats.Conn, error) {
	var natsOpts = []nats.Option{nats.Name("Auditor")}

	natsOpts = append(natsOpts, nats.UserCredentials(creds))
	nc, err := nats.Connect(server, natsOpts...)
	if err != nil {
		return nil, nil, err
	}

	sender, err := cnats.NewSenderFromConn(nc, subject)
	if err != nil {
		return nil, nil, err
	}

	cloudEventsClient, err := cloudevents.NewClient(sender)
	if err != nil {
		return nil, nil, err
	}

	return cloudEventsClient, nc, nil
}

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

package receiver

import (
	"context"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	eventz "github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	Subject = "ClusterEvents"
)

func PublishEvent(client cloudevents.Client, obj interface{}) error {
	event := cloudevents.NewEvent()
	setEventDefaults(&event)
	data, err := obj.(*unstructured.Unstructured).MarshalJSON()
	if err != nil {
		return err
	}
	if err = event.SetData("application/json", data); err != nil {
		return err
	}
	result := client.Send(context.Background(), event)
	if cloudevents.IsUndelivered(result) {
		log.Printf("failed to send: %v", err)
		return err
	}
	log.Printf("Published event to channel `%s` and acknoledged: %v", Subject, cloudevents.IsACK(result))

	return err
}

func setEventDefaults(event *eventz.Event) {
	event.SetID(uuid.New().String())
	event.SetSubject(Subject)
	event.SetType("cluster.event")
	event.SetSource("kubeshield.dev/auditor")
	event.SetTime(time.Now())
}

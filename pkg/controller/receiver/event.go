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
)

const (
	Subject = "ClusterEvents"
)

// PublishEvent sends the events to receiver server
func PublishEvent(client cloudevents.Client, op string, obj []byte) error {
	event := cloudevents.NewEvent()
	setEventDefaults(&event, op)
	if err := event.SetData("application/json", obj); err != nil {
		return err
	}
	result := client.Send(context.Background(), event)
	if cloudevents.IsUndelivered(result) {
		log.Printf("failed to send: %v", result.Error())
		return result
	}
	log.Printf("Published event to channel `%s` and acknoledged: %v", Subject, cloudevents.IsACK(result))

	return nil
}

func setEventDefaults(event *eventz.Event, op string) {
	event.SetID(uuid.New().String())
	event.SetSubject(Subject)
	event.SetType(op)
	event.SetSource("kubeshield.dev/auditor")
	event.SetTime(time.Now())
}

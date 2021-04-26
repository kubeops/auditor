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
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding/format"
	eventz "github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
)

const (
	EventCreate = "create"
	EventUpdate = "update"
	EventDelete = "delete"
)

type OperatorEvent struct {
	Resource   *unstructured.Unstructured `json:"resource"`
	ResourceID v1alpha1.ResourceID        `json:"resourceID"`
	LicenseID  string                     `json:"licenseID"`
}

// PublishEvent sends the events to receiver server
func PublishEvent(nc *nats.Conn, natsSubject string, op string, obj interface{}) error {
	event := cloudevents.NewEvent()
	setEventDefaults(&event, natsSubject, op)

	if err := event.SetData(eventz.ApplicationJSON, obj); err != nil {
		return err
	}

	data, err := format.JSON.Marshal(&event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel()

	for {
		_, err = nc.Request(natsSubject, data, time.Second*5)
		if err == nil {
			cancel()
		} else {
			klog.Warningln(err)
		}

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				klog.Warningf("failed to send event : %s", string(data))
			} else if ctx.Err() == context.Canceled {
				klog.Infof("Published event `%s` to channel `%s` and acknoledged", op, natsSubject)
			}
			return nil
		default:
			time.Sleep(time.Microsecond * 100)
		}
	}
}

func setEventDefaults(event *eventz.Event, natsSubject, op string) {
	event.SetID(uuid.New().String())
	event.SetSubject(natsSubject)
	event.SetType(op)
	event.SetSource("kubeops.dev/auditor")
	event.SetTime(time.Now())
}

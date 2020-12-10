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

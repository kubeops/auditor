package usage

import (
	"fmt"
	"time"

	"github.com/nats-io/jsm.go"
	natsio "github.com/nats-io/nats.go"
)

func (opts *NatsOptions) StartJSSubscriber() {
	_, mgr, err := Connect(opts)
	if err != nil {
		return
	}

	ticker := time.NewTicker(time.Minute * 10)
	for {
		select {
		case <-ticker.C:
			msg, err := mgr.NextMsg(opts.JSStream, opts.JSConsumer)
			if err != nil {
				return
			}

			// Process Message
			fmt.Println(msg)
		}
	}
}

func Connect(opts *NatsOptions) (*natsio.Conn, *jsm.Manager, error) {
	nc, err := natsio.Connect(opts.JetstreamServer, natsio.UserCredentials(opts.JSCredentialFile))
	if err != nil {
		return nil, nil, err
	}
	mgr, err := jsm.New(nc, jsm.WithTimeout(time.Second))
	if err != nil {
		return nil, nil, err
	}

	if err = EnsureStreamExists(opts.JSStream, opts.JSSubject, mgr); err != nil {
		return nil, nil, err
	}

	if err = EnsureConsumerExists(opts.JSConsumer, opts.JSSubject, opts.JSStream, mgr); err != nil {
		return nil, nil, err
	}

	return nc, mgr, nil
}

func EnsureStreamExists(name, subject string, mgr *jsm.Manager) error {
	_, err := mgr.LoadOrNewStreamFromDefault(name, jsm.DefaultWorkQueue,
		jsm.Subjects(subject), jsm.FileStorage(), jsm.MaxAge(24*365*time.Hour),
		jsm.DiscardOld(), jsm.MaxMessages(-1), jsm.MaxBytes(-1), jsm.MaxMessageSize(512),
		jsm.DuplicateWindow(1*time.Hour))
	if err != nil {
		return err
	}

	return nil
}

func EnsureConsumerExists(name, filter, stream string, mgr *jsm.Manager) error {
	_, err := mgr.NewConsumerFromDefault(stream, jsm.DefaultConsumer,
		jsm.DurableName(name), jsm.MaxDeliveryAttempts(5),
		jsm.AckWait(30*time.Second), jsm.AcknowledgeExplicit(), jsm.ReplayInstantly(),
		jsm.DeliverAllAvailable(), jsm.FilterStreamBySubject(filter))
	if err != nil {
		return err
	}

	return nil
}

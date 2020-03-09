package nats

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/pkg/event"

	context2 "github.com/cloudevents/sdk-go/pkg/cloudevents/context"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/transport"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Transport adheres to transport.Transport.
var _ transport.Transport = (*Transport)(nil)

const (
	// TransportName is the name of this transport.
	TransportName = "NATS"
)

// Transport acts as both a NATS client and a NATS handler.
type Transport struct {
	Encoding    Encoding
	Conn        *nats.Conn
	ConnOptions []nats.Option
	NatsURL     string
	Subject     string

	sub *nats.Subscription

	Receiver transport.Receiver
	// Converter is invoked if the incoming transport receives an undecodable
	// message.
	Converter transport.Converter

	codec transport.Codec
}

// New creates a new NATS transport.
func New(natsURL, subject string, opts ...Option) (*Transport, error) {
	t := &Transport{
		Subject:     subject,
		NatsURL:     natsURL,
		ConnOptions: []nats.Option{},
	}

	err := t.applyOptions(opts...)
	if err != nil {
		return nil, err
	}

	err = t.connect()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Transport) connect() error {
	var err error

	t.Conn, err = nats.Connect(t.NatsURL, t.ConnOptions...)

	return err
}

func (t *Transport) applyOptions(opts ...Option) error {
	for _, fn := range opts {
		if err := fn(t); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transport) loadCodec() bool {
	if t.codec == nil {
		switch t.Encoding {
		case Default:
			t.codec = &Codec{}
		case StructuredV03:
			t.codec = &CodecV03{Encoding: t.Encoding}
		case StructuredV1:
			t.codec = &CodecV1{Encoding: t.Encoding}
		default:
			return false
		}
	}
	return true
}

// Send implements Transport.Send
func (t *Transport) Send(ctx context.Context, event event.Event) error {
	// TODO populate response context properly.
	if ok := t.loadCodec(); !ok {
		return fmt.Errorf("unknown encoding set on transport: %d", t.Encoding)
	}

	msg, err := t.codec.Encode(ctx, event)
	if err != nil {
		return err
	}

	if m, ok := msg.(*Message); ok {
		return t.Conn.Publish(t.Subject, m.Body)
	}

	return fmt.Errorf("failed to encode Event into a Message")
}

// Request implements Transport.Request
func (t *Transport) Request(ctx context.Context, event event.Event) (*event.Event, error) {
	return nil, fmt.Errorf("Transport.Request is not supported for NATS")
}

// SetReceiver implements Transport.SetReceiver
func (t *Transport) SetReceiver(r transport.Receiver) {
	t.Receiver = r
}

// SetConverter implements Transport.SetConverter
func (t *Transport) SetConverter(c transport.Converter) {
	t.Converter = c
}

// HasConverter implements Transport.HasConverter
func (t *Transport) HasConverter() bool {
	return t.Converter != nil
}

// StartReceiver implements Transport.StartReceiver
// NOTE: This is a blocking call.
func (t *Transport) StartReceiver(ctx context.Context) (err error) {
	if t.Conn == nil {
		return fmt.Errorf("no active nats connection")
	}
	if t.sub != nil {
		return fmt.Errorf("already subscribed")
	}
	if ok := t.loadCodec(); !ok {
		return fmt.Errorf("unknown encoding set on transport: %d", t.Encoding)
	}

	// TODO: there could be more than one subscription. Might have to do a map
	// of subject to subscription.

	if t.Subject == "" {
		return fmt.Errorf("subject required for nats listen")
	}

	// Simple Async Subscriber
	t.sub, err = t.Conn.Subscribe(t.Subject, func(m *nats.Msg) {
		logger := context2.LoggerFrom(ctx)
		msg := &Message{
			Body: m.Data,
		}
		e, err := t.codec.Decode(ctx, msg)
		// If codec returns and error, try with the converter if it is set.
		if err != nil && t.HasConverter() {
			e, err = t.Converter.Convert(ctx, msg, err)
		}
		if err != nil {
			logger.Errorw("failed to decode message", zap.Error(err)) // TODO: create an error channel to pass this up
			return
		}
		// TODO: I do not know enough about NATS to implement reply.
		// For now, NATS does not support reply.
		if err := t.Receiver.Receive(context.TODO(), *e, nil); err != nil {
			logger.Warnw("nats receiver return err", zap.Error(err))
		}
	})
	defer func() {
		if t.sub != nil {
			err2 := t.sub.Unsubscribe()
			if err != nil {
				err = err2 // Set the returned error if not already set.
			}
			t.sub = nil
		}
	}()
	<-ctx.Done()
	return err
}

// HasTracePropagation implements Transport.HasTracePropagation
func (t *Transport) HasTracePropagation() bool {
	return false
}

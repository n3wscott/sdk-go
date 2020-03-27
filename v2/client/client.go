package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"go.uber.org/zap"

	"github.com/cloudevents/sdk-go/v2/binding"
	cecontext "github.com/cloudevents/sdk-go/v2/context"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/cloudevents/sdk-go/v2/protocol"
)

// Client interface defines the runtime contract the CloudEvents client supports.
type Client interface {
	// Send will transmit the given event over the client's configured transport.
	Send(ctx context.Context, event event.Event) protocol.Result

	// Request will transmit the given event over the client's configured
	// transport and return any response event.
	Request(ctx context.Context, event event.Event) (*event.Event, protocol.Result)

	// StartReceiver will register the provided function for callback on receipt
	// of a cloudevent. It will also start the underlying protocol as it has
	// been configured.
	// This call is blocking.
	// Valid fn signatures are:
	// * func()
	// * func() error
	// * func(context.Context)
	// * func(context.Context) protocol.Result
	// * func(event.Event)
	// * func(event.Event) protocol.Result
	// * func(context.Context, event.Event)
	// * func(context.Context, event.Event) protocol.Result
	// * func(event.Event) *event.Event
	// * func(event.Event) (*event.Event, protocol.Result)
	// * func(context.Context, event.Event) *event.Event
	// * func(context.Context, event.Event) (*event.Event, protocol.Result)
	StartReceiver(ctx context.Context, fn interface{}) error
}

// New produces a new client with the provided transport object and applied
// client options.
func New(obj interface{}, opts ...Option) (Client, error) {
	c := &ceClient{}

	if p, ok := obj.(protocol.Sender); ok {
		c.sender = p
	}
	if p, ok := obj.(protocol.Requester); ok {
		c.requester = p
	}
	if p, ok := obj.(protocol.Responder); ok {
		c.responder = p
	}
	if p, ok := obj.(protocol.Receiver); ok {
		c.receiver = p
	}
	if p, ok := obj.(protocol.Opener); ok {
		c.opener = p
	}

	if err := c.applyOptions(opts...); err != nil {
		return nil, err
	}
	return c, nil
}

type ceClient struct {
	sender    protocol.Sender
	requester protocol.Requester
	receiver  protocol.Receiver
	responder protocol.Responder
	// Optional.
	opener protocol.Opener

	outboundContextDecorators []func(context.Context) context.Context
	invoker                   Invoker
	receiverMu                sync.Mutex
	eventDefaulterFns         []EventDefaulter
}

func (c *ceClient) applyOptions(opts ...Option) error {
	for _, fn := range opts {
		if err := fn(c); err != nil {
			return err
		}
	}
	return nil
}

func (c *ceClient) Send(ctx context.Context, e event.Event) protocol.Result {
	if c.sender == nil {
		return errors.New("sender not set")
	}

	for _, f := range c.outboundContextDecorators {
		ctx = f(ctx)
	}

	if len(c.eventDefaulterFns) > 0 {
		for _, fn := range c.eventDefaulterFns {
			e = fn(ctx, e)
		}
	}

	if err := e.Validate(); err != nil {
		return err
	}

	return c.sender.Send(ctx, (*binding.EventMessage)(&e))
}

func (c *ceClient) Request(ctx context.Context, e event.Event) (*event.Event, protocol.Result) {
	if c.requester == nil {
		return nil, errors.New("requester not set")
	}
	for _, f := range c.outboundContextDecorators {
		ctx = f(ctx)
	}

	if len(c.eventDefaulterFns) > 0 {
		for _, fn := range c.eventDefaulterFns {
			e = fn(ctx, e)
		}
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	// If provided a requester, use it to do request/response.
	var resp *event.Event
	msg, err := c.requester.Request(ctx, (*binding.EventMessage)(&e))
	defer func() {
		if msg == nil {
			return
		}
		if err := msg.Finish(err); err != nil {
			cecontext.LoggerFrom(ctx).Warnw("failed calling message.Finish", zap.Error(err))
		}
	}()

	if msg != nil {
		fmt.Println("message was not nil")
	}
	//fmt.Printf("%#v", msg)
	// try to turn msg into an event, it might not work and that is ok.
	if rs, err := binding.ToEvent(ctx, msg); err != nil {
		cecontext.LoggerFrom(ctx).Debugw("failed calling ToEvent", zap.Error(err), zap.Any("resp", msg))
	} else {
		resp = rs
	}

	return resp, err
}

// StartReceiver sets up the given fn to handle Receive.
// See Client.StartReceiver for details. This is a blocking call.
func (c *ceClient) StartReceiver(ctx context.Context, fn interface{}) error {
	c.receiverMu.Lock()
	defer c.receiverMu.Unlock()

	if c.invoker != nil {
		return fmt.Errorf("client already has a receiver")
	}

	invoker, err := newReceiveInvoker(fn, c.eventDefaulterFns...) // TODO: this will have to pick between a observed invoker or not.
	if err != nil {
		return err
	}
	if invoker.IsReceiver() && c.receiver == nil {
		return fmt.Errorf("mismatched receiver callback without protocol.Receiver supported by protocol")
	}
	if invoker.IsResponder() && c.responder == nil {
		return fmt.Errorf("mismatched receiver callback without protocol.Responder supported by protocol")
	}
	c.invoker = invoker

	defer func() {
		c.invoker = nil
	}()

	// Start the opener, if set.
	if c.opener != nil {
		go func() {
			// TODO: handle error correctly here.
			if err := c.opener.OpenInbound(ctx); err != nil {
				panic(err)
			}
		}()
	}

	var msg binding.Message
	var respFn protocol.ResponseFn
	// Start Polling.
	for {
		if c.responder != nil {
			msg, respFn, err = c.responder.Respond(ctx)
		} else if c.receiver != nil {
			msg, err = c.receiver.Receive(ctx)
		} else {
			return errors.New("responder nor receiver set")
		}

		if err == io.EOF { // Normal close
			return nil
		}
		//else if err != nil {
		//	return err
		//}
		if err := c.invoker.Invoke(ctx, msg, respFn); err != nil {
			return err
		}
	}
}

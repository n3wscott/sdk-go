package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudevents/sdk-go/pkg/cloudevents"
	"github.com/cloudevents/sdk-go/pkg/cloudevents/types"
	"reflect"
)

// Receive is the signature of a fn to be invoked for incoming cloudevents.
// If fn returns an error, EventResponse will not be considered by the client or
// or transport.
// This is just an FYI:
type ReceiveFull func(context.Context, cloudevents.Event, *cloudevents.EventResponse) error

type receiverFn struct {
	numIn   int
	fnValue reflect.Value

	hasContextIn       bool
	hasEventIn         bool
	hasDataIn          bool
	hasEventResponseIn bool

	dataType reflect.Type

	hasErrorOut bool
}

const (
	inParamUsage  = "expected a function taking either no parameters, one or more of (context.Context, cloudevents.Event, *cloudevents.EventResponse) ordered"
	outParamUsage = "expected a function returning either nothing or an error"
)

var (
	contextType          = reflect.TypeOf((*context.Context)(nil)).Elem()
	eventType            = reflect.TypeOf((*cloudevents.Event)(nil)).Elem()
	eventPtrType         = reflect.TypeOf((*cloudevents.Event)(nil)) // want the ptr type
	eventResponseType    = reflect.TypeOf((*cloudevents.EventResponse)(nil)).Elem()
	eventResponsePtrType = reflect.TypeOf((*cloudevents.EventResponse)(nil)) // want the ptr type
	errorType            = reflect.TypeOf((*error)(nil)).Elem()
)

// receiver creates a receiverFn wrapper class that is used by the client to
// validate and invoke the provided function. Parameter order matters.
// Any represents an object be automatically called with event.DataAs(&any)
// Valid fn signatures are:
// choose none,
// * func()
// * func() error
// choose one:[context.Context, cloudevents.Event, Any, *cloudevents.EventResponse] = {1}
// * func({1})
// * func({1}) error
// choose two:[context.Context, cloudevents.Event, Any, *cloudevents.EventResponse] = {1}, {2}
// * func({1}, {2})
// * func({1}, {2}) error
// choose three:[context.Context, cloudevents.Event, Any, *cloudevents.EventResponse] = {1}, {2}, {3}
// * func({1}, {2}, {3})
// * func({1}, {2}, {3}) error
// choose all,
// * func(context.Context, cloudevents.Event, Any, *cloudevents.EventResponse)
// * func(context.Context, cloudevents.Event, Any, *cloudevents.EventResponse) error
//
func receiver(fn interface{}) (*receiverFn, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, errors.New("must pass a function to handle events")
	}

	r := &receiverFn{
		fnValue: reflect.ValueOf(fn),
		numIn:   fnType.NumIn(),
	}
	if err := r.validate(fnType); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *receiverFn) invoke(ctx context.Context, event cloudevents.Event, resp *cloudevents.EventResponse) error {
	args := make([]reflect.Value, 0, r.numIn)

	if r.numIn > 0 {
		if r.hasContextIn {
			args = append(args, reflect.ValueOf(ctx))
		}
		if r.hasEventIn {
			args = append(args, reflect.ValueOf(event))
		}
		if r.hasDataIn {
			data, _ := types.AllocateFromType(r.dataType)
			if err := event.DataAs(&data); err != nil {
				return fmt.Errorf("failed to get dataAs: %s", err.Error())
			}
			args = append(args, reflect.ValueOf(data))
		}
		if r.hasEventResponseIn {
			args = append(args, reflect.ValueOf(resp))
		}
	}
	v := r.fnValue.Call(args)
	if r.hasErrorOut && len(v) >= 1 {
		if err, ok := v[0].Interface().(error); ok {
			return err
		}
	}
	return nil
}

func unconvertibleToEventResponseType(t reflect.Type) bool {
	return !t.ConvertibleTo(eventResponsePtrType)
}

func unconvertibleToEventType(t reflect.Type) bool {
	return !t.ConvertibleTo(eventType)
}

func unconvertibleToContextType(t reflect.Type) bool {
	return !t.ConvertibleTo(contextType)
}

func unconvertibleToAny(t reflect.Type) bool {
	return t.ConvertibleTo(contextType) ||
		t.ConvertibleTo(eventType) || t.ConvertibleTo(eventPtrType) ||
		t.ConvertibleTo(eventResponseType) || t.ConvertibleTo(eventResponsePtrType)
}

// Verifies that the inputs to a function have a valid signature
// Valid input is to be [0, all] of
// context.Context, cloudevents.Event, *data, *cloudevents.EventResponse in this order.
func (r *receiverFn) validateInParamSignature(fnType reflect.Type) error {
	r.hasContextIn = false
	r.hasEventIn = false
	r.hasDataIn = false
	r.hasEventResponseIn = false

	switch fnType.NumIn() {
	case 4:
		t := fnType.In(3)
		if unconvertibleToEventResponseType(t) {
			return fmt.Errorf("%s; cannot convert parameter 4 from %s to *cloudevents.EventResponse", inParamUsage, t)
		} else {
			r.hasEventResponseIn = true
		}

		t = fnType.In(2)
		if unconvertibleToAny(t) {
			return fmt.Errorf("%s; cannot convert parameter 3 from %s to Any", inParamUsage, t)
		} else {
			r.dataType = t
			r.hasDataIn = true
		}

		t = fnType.In(1)
		if unconvertibleToEventType(t) {
			return fmt.Errorf("%s; cannot convert parameter 2 from %s to cloudevents.Event", inParamUsage, t)
		} else {
			r.hasEventIn = true
		}

		t = fnType.In(0)
		if unconvertibleToContextType(t) {
			return fmt.Errorf("%s; cannot convert parameter 1 from %s to context.Context", inParamUsage, t)
		} else {
			r.hasContextIn = true
		}
		return nil

	case 3:
		// has to be cloudevents.Event, *cloudevents.EventResponse
		t := fnType.In(2)
		if unconvertibleToEventResponseType(t) {
			return fmt.Errorf("%s; cannot convert parameter 2 from %s to *cloudevents.EventResponse", inParamUsage, t)
		} else {
			r.hasEventResponseIn = true
		}
		fallthrough
	case 2:
		// can be cloudevents.Event or *cloudevents.EventResponse
		t := fnType.In(1)
		if unconvertibleToEventResponseType(t) {
			if unconvertibleToEventType(t) {
				return fmt.Errorf("%s; cannot convert parameter 1 from %s to cloudevents.Event or *cloudevents.EventResponse", inParamUsage, t)
			} else {
				r.hasEventIn = true
			}
		} else if r.hasEventResponseIn {
			return fmt.Errorf("%s; duplicate parameter of type *cloudevents.EventResponse", inParamUsage)
		} else {
			r.hasEventResponseIn = true
		}
		fallthrough
	case 1:
		t := fnType.In(0)
		if unconvertibleToContextType(t) {
			if unconvertibleToEventResponseType(t) {
				if unconvertibleToEventType(t) {
					return fmt.Errorf("%s; cannot convert parameter 0 from %s to context.Context, cloudevents.Event or *cloudevents.EventResponse", inParamUsage, t)
				} else if r.hasEventIn {
					return fmt.Errorf("%s; duplicate parameter of type cloudevents.Event", inParamUsage)
				} else {
					r.hasEventIn = true
				}
			} else if r.hasEventResponseIn {
				return fmt.Errorf("%s; duplicate parameter of type *cloudevents.EventResponse", inParamUsage)
			} else if r.hasEventIn {
				return fmt.Errorf("%s; out of order parameter 0 for %s", inParamUsage, t)
			} else {
				r.hasEventResponseIn = true
			}
		} else {
			r.hasContextIn = true
		}
		fallthrough
	case 0:
		return nil
	default:
		return fmt.Errorf("%s; function has too many parameters (%d)", inParamUsage, fnType.NumIn())
	}
}

// Verifies that the outputs of a function have a valid signature
// Valid output signatures:
// (), (error)
func (r *receiverFn) validateOutParamSignature(fnType reflect.Type) error {
	r.hasErrorOut = false
	switch fnType.NumOut() {
	case 1:
		paramNo := fnType.NumOut() - 1
		paramType := fnType.Out(paramNo)
		if !paramType.ConvertibleTo(errorType) {
			return fmt.Errorf("%s; cannot convert return type %d from %s to error", outParamUsage, paramNo, paramType)
		} else {
			r.hasErrorOut = true
		}
		fallthrough
	case 0:
		return nil
	default:
		return fmt.Errorf("%s; function has too many return types (%d)", outParamUsage, fnType.NumOut())
	}
}

// validateReceiverFn validates that a function has the right number of in and
// out params and that they are of allowed types.
func (r *receiverFn) validate(fnType reflect.Type) error {
	if err := r.validateInParamSignature(fnType); err != nil {
		return err
	}
	if err := r.validateOutParamSignature(fnType); err != nil {
		return err
	}
	return nil
}

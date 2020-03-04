package http

import (
	"context"

	"github.com/cloudevents/sdk-go/pkg/event"

	cecontext "github.com/cloudevents/sdk-go/pkg/cloudevents/context"
)

// Encoding to use for HTTP transport.
type Encoding int32

type EncodingSelector func(context.Context, event.Event) Encoding

const (
	// Default
	Default Encoding = iota
	BinaryV03
	// StructuredV03 is Structured CloudEvents spec v0.3.
	StructuredV03
	// BatchedV03 is Batched CloudEvents spec v0.3.
	BatchedV03
	// BinaryV1 is Binary CloudEvents spec v1.0.
	BinaryV1
	// StructuredV03 is Structured CloudEvents spec v1.0.
	StructuredV1
	// BatchedV1 is Batched CloudEvents spec v1.0.
	BatchedV1

	// Unknown is unknown.
	Unknown

	// Binary is used for Context Based Encoding Selections to use the
	// DefaultBinaryEncodingSelectionStrategy
	Binary = "binary"

	// Structured is used for Context Based Encoding Selections to use the
	// DefaultStructuredEncodingSelectionStrategy
	Structured = "structured"

	// Batched is used for Context Based Encoding Selections to use the
	// DefaultStructuredEncodingSelectionStrategy
	Batched = "batched"
)

func ContextBasedEncodingSelectionStrategy(ctx context.Context, e event.Event) Encoding {
	encoding := cecontext.EncodingFrom(ctx)
	switch encoding {
	case "", Binary:
		return DefaultBinaryEncodingSelectionStrategy(ctx, e)
	case Structured:
		return DefaultStructuredEncodingSelectionStrategy(ctx, e)
	}
	return Default
}

// DefaultBinaryEncodingSelectionStrategy implements a selection process for
// which binary encoding to use based on spec version of the event.
func DefaultBinaryEncodingSelectionStrategy(ctx context.Context, e event.Event) Encoding {
	switch e.SpecVersion() {
	case event.CloudEventsVersionV03:
		return BinaryV03
	case event.CloudEventsVersionV1:
		return BinaryV1
	}
	// Unknown version, return Default.
	return Default
}

// DefaultStructuredEncodingSelectionStrategy implements a selection process
// for which structured encoding to use based on spec version of the event.
func DefaultStructuredEncodingSelectionStrategy(ctx context.Context, e event.Event) Encoding {
	switch e.SpecVersion() {
	case event.CloudEventsVersionV03:
		return StructuredV03
	case event.CloudEventsVersionV1:
		return StructuredV1
	}
	// Unknown version, return Default.
	return Default
}

// String pretty-prints the encoding as a string.
func (e Encoding) String() string {
	switch e {
	case Default:
		return "Default Encoding " + e.Version()

	// Binary
	case BinaryV03, BinaryV1:
		return "Binary Encoding " + e.Version()

	// Structured
	case StructuredV03, StructuredV1:
		return "Structured Encoding " + e.Version()

	// Batched
	case BatchedV03, BatchedV1:
		return "Batched Encoding " + e.Version()

	default:
		return "Unknown Encoding"
	}
}

// Version pretty-prints the encoding version as a string.
func (e Encoding) Version() string {
	switch e {
	case Default:
		return "Default"

	// Version 0.3
	case BinaryV03, StructuredV03, BatchedV03:
		return "v0.3"

	// Version 1.0
	case BinaryV1, StructuredV1, BatchedV1:
		return "v1.0"

	// Unknown
	default:
		return "Unknown"
	}
}

// Codec creates a structured string to represent the the codec version.
func (e Encoding) Codec() string {
	switch e {
	case Default:
		return "default"

	// Version 0.3
	case BinaryV03:
		return "binary/v0.3"
	case StructuredV03:
		return "structured/v0.3"
	case BatchedV03:
		return "batched/v0.3"

	// Version 1.0
	case BinaryV1:
		return "binary/v1.0"
	case StructuredV1:
		return "structured/v1.0"
	case BatchedV1:
		return "batched/v1.0"

	// Unknown
	default:
		return "unknown"
	}
}

// Name creates a string to represent the the codec name.
func (e Encoding) Name() string {
	switch e {
	case Default:
		return Binary
	case BinaryV03, BinaryV1:
		return Binary
	case StructuredV03, StructuredV1:
		return Structured
	case BatchedV03, BatchedV1:
		return Batched
	default:
		return Binary
	}
}

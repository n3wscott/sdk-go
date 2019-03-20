package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// TransportContext allows a Receiver to understand the context of a request.
type TransportContext struct {
	URI    string
	Host   string
	Method string
	Header http.Header

	// This is e
	IgnoreHeaderPrefixes []string
}

func NewTransportContext(req *http.Request) TransportContext {
	var tx *TransportContext
	if req != nil {
		tx = &TransportContext{
			URI:    req.RequestURI,
			Host:   req.Host,
			Method: req.Method,
			Header: req.Header,
		}
	} else {
		tx = &TransportContext{}
	}
	tx.AddIgnoreHeaderPrefix("accept-encoding", "user-agent", "connection", "content-type")
	return *tx
}

// AttendToHeaders returns the list of headers that exist in the TransportContext that are not currently in
// tx.IgnoreHeaderPrefix.
func (tx TransportContext) AttendToHeaders() []string {
	a := []string(nil)
	if tx.Header != nil && len(tx.Header) > 0 {
		for k := range tx.Header {
			if tx.shouldIgnoreHeader(k) {
				continue
			}
			a = append(a, k)
		}
	}
	return a
}

func (tx TransportContext) shouldIgnoreHeader(h string) bool {
	for _, v := range tx.IgnoreHeaderPrefixes {
		if strings.HasPrefix(strings.ToLower(h), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func (tx TransportContext) String() string {
	b := strings.Builder{}

	b.WriteString("Transport Context,\n")
	b.WriteString("  URI: " + tx.URI + "\n")
	b.WriteString("  Host: " + tx.Host + "\n")
	b.WriteString("  Method: " + tx.Method + "\n")

	if tx.Header != nil && len(tx.Header) > 0 {
		b.WriteString("  Header:\n")
		for _, k := range tx.AttendToHeaders() {
			b.WriteString(fmt.Sprintf("    %s: %s\n", k, tx.Header.Get(k)))
		}
	}

	return b.String()
}

// AddIgnoreHeaderPrefix controls what header key is to be attended to and/or printed.
func (tx *TransportContext) AddIgnoreHeaderPrefix(prefix ...string) {
	if tx.IgnoreHeaderPrefixes == nil {
		tx.IgnoreHeaderPrefixes = []string(nil)
	}
	tx.IgnoreHeaderPrefixes = append(tx.IgnoreHeaderPrefixes, prefix...)
}

// Opaque key type used to store TransportContext
type transportContextKeyType struct{}

var transportContextKey = transportContextKeyType{}

// WithTransportContext return a context with the given TransportContext into the provided context object.
func WithTransportContext(ctx context.Context, tcxt TransportContext) context.Context {
	return context.WithValue(ctx, transportContextKey, tcxt)
}

func TransportContextFrom(ctx context.Context) TransportContext {
	tctx := ctx.Value(transportContextKey)
	if tctx != nil {
		if tx, ok := tctx.(TransportContext); ok {
			return tx
		}
		if tx, ok := tctx.(*TransportContext); ok {
			return *tx
		}
	}
	return TransportContext{}
}

// Opaque key type used to store Headers
type headerKeyType struct{}

var headerKey = headerKeyType{}

// ContextWithHeader returns a context with a header added to the given context.
// Can be called multiple times to set multiple header key/value pairs.
func ContextWithHeader(ctx context.Context, key, value string) context.Context {
	header := HeaderFrom(ctx)
	header.Add(key, value)
	return context.WithValue(ctx, headerKey, header)
}

// HeaderFrom extracts the header oject in the given context. Always returns a non-nil Header.
func HeaderFrom(ctx context.Context) http.Header {
	header := ctx.Value(headerKey)
	if header != nil {
		if h, ok := header.(http.Header); ok {
			return h
		}
	}
	return http.Header{}
}

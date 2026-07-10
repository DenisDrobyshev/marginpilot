package app

import (
	"context"
	"errors"
)

// ErrKeyNotFound is returned by a CallerResolver when the API key is unknown.
var ErrKeyNotFound = errors.New("api key not found")

// CallerResolver maps an opaque virtual API key to the caller behind it. The
// identity gRPC adapter implements this; when the gateway is started without an
// identity service, the HTTP handler falls back to reading caller headers.
type CallerResolver interface {
	Resolve(ctx context.Context, apiKey string) (Caller, error)
}

// Package logger builds the structured logger used across all services.
package logger

import (
	"log/slog"
	"os"
)

// Logger is an alias for slog.Logger, so call sites can depend on this package
// without importing log/slog directly.
type Logger = slog.Logger

// New returns a JSON structured logger tagged with the service name.
func New(service string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h).With("service", service)
}

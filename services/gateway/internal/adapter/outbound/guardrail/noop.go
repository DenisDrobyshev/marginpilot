// Package guardrail contains outbound adapters implementing port.Guardrail.
package guardrail

import "github.com/marginpilot/gateway/internal/domain"

// Noop lets every request through unchanged.
type Noop struct{}

// NewNoop constructs the no-op guardrail.
func NewNoop() Noop { return Noop{} }

// Check returns the request unchanged.
func (Noop) Check(req domain.ChatRequest) (domain.ChatRequest, error) { return req, nil }

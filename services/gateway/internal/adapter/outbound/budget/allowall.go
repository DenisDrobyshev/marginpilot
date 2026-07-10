// Package budget contains outbound adapters implementing port.BudgetChecker.
package budget

import "context"

// AllowAll is a permissive checker for local development. The production
// implementation calls the Budget/Policy service (Redis-backed token buckets
// and hierarchical limits) over gRPC and must stay within a few milliseconds
// to remain viable on the request hot path.
type AllowAll struct{}

// NewAllowAll constructs the permissive checker.
func NewAllowAll() *AllowAll { return &AllowAll{} }

// Allow always permits the call in this stub.
func (a *AllowAll) Allow(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}

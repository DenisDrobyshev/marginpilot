// Package enforcer holds the budget decision logic, independent of Redis or
// gRPC. It depends only on the Store port, so it is unit-tested with a fake.
package enforcer

import "context"

// Store is the persistence port the enforcer needs.
type Store interface {
	// Spend returns the customer's accumulated spend this period, in micros.
	Spend(ctx context.Context, tenant, customer string) (int64, error)
	// Limit returns the customer's configured limit in micros, or <=0 if unset.
	Limit(ctx context.Context, tenant, customer string) (int64, error)
	// SetLimit persists a customer's limit in micros.
	SetLimit(ctx context.Context, tenant, customer string, micros int64) error
	// AllowRate reports whether the customer is under the request-rate cap.
	AllowRate(ctx context.Context, tenant, customer string) (bool, error)
}

// Decision is the outcome of an Allow check.
type Decision struct {
	Allowed         bool
	RemainingMicros int64
	Reason          string // "", "budget" or "rate_limit"
}

// Enforcer applies rate limits and spend budgets.
type Enforcer struct {
	store        Store
	defaultLimit int64
}

// New builds an enforcer with a fallback limit (micros) for customers that
// have no explicit limit configured.
func New(store Store, defaultLimitMicros int64) *Enforcer {
	return &Enforcer{store: store, defaultLimit: defaultLimitMicros}
}

// Allow decides whether the customer may make another billable call. Rate
// limiting is checked first (cheapest, protects against runaway loops), then
// the spend budget.
func (e *Enforcer) Allow(ctx context.Context, tenant, customer string) (Decision, error) {
	okRate, err := e.store.AllowRate(ctx, tenant, customer)
	if err != nil {
		return Decision{}, err
	}

	limit, err := e.store.Limit(ctx, tenant, customer)
	if err != nil {
		return Decision{}, err
	}
	if limit <= 0 {
		limit = e.defaultLimit
	}

	spent, err := e.store.Spend(ctx, tenant, customer)
	if err != nil {
		return Decision{}, err
	}
	remaining := limit - spent

	switch {
	case !okRate:
		return Decision{Allowed: false, RemainingMicros: remaining, Reason: "rate_limit"}, nil
	case remaining <= 0:
		return Decision{Allowed: false, RemainingMicros: remaining, Reason: "budget"}, nil
	default:
		return Decision{Allowed: true, RemainingMicros: remaining}, nil
	}
}

// SetLimit persists a customer's spend limit.
func (e *Enforcer) SetLimit(ctx context.Context, tenant, customer string, micros int64) error {
	return e.store.SetLimit(ctx, tenant, customer, micros)
}

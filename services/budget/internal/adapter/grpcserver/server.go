// Package grpcserver adapts the enforcer to the BudgetService gRPC contract and
// emits an alert whenever a request is blocked.
package grpcserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	budgetv1 "github.com/marginpilot/contracts/gen/budget/v1"
	"github.com/marginpilot/shared/events"

	"github.com/marginpilot/budget/internal/enforcer"
)

// AlertPublisher publishes alert events. A nil publisher disables alerting.
type AlertPublisher interface {
	Publish(ctx context.Context, e events.AlertEvent) error
}

// Server implements budgetv1.BudgetServiceServer.
type Server struct {
	budgetv1.UnimplementedBudgetServiceServer
	enf     *enforcer.Enforcer
	alerter AlertPublisher
	log     *slog.Logger
}

// New wraps an enforcer as a gRPC server. alerter may be nil.
func New(enf *enforcer.Enforcer, alerter AlertPublisher, log *slog.Logger) *Server {
	return &Server{enf: enf, alerter: alerter, log: log}
}

// Allow answers the gateway's hot-path budget check and alerts on a block.
func (s *Server) Allow(ctx context.Context, req *budgetv1.AllowRequest) (*budgetv1.AllowResponse, error) {
	d, err := s.enf.Allow(ctx, req.GetTenantId(), req.GetCustomerId())
	if err != nil {
		return nil, err
	}
	if !d.Allowed && s.alerter != nil {
		s.emitAlert(req.GetTenantId(), req.GetCustomerId(), d.Reason)
	}
	return &budgetv1.AllowResponse{
		Allowed:         d.Allowed,
		RemainingMicros: d.RemainingMicros,
		Reason:          d.Reason,
	}, nil
}

// SetLimit configures a customer's spend limit.
func (s *Server) SetLimit(ctx context.Context, req *budgetv1.SetLimitRequest) (*budgetv1.SetLimitResponse, error) {
	if err := s.enf.SetLimit(ctx, req.GetTenantId(), req.GetCustomerId(), req.GetLimitMicros()); err != nil {
		return nil, err
	}
	return &budgetv1.SetLimitResponse{Ok: true}, nil
}

// emitAlert fires the alert asynchronously so it never slows the hot path.
func (s *Server) emitAlert(tenant, customer, reason string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		e := events.AlertEvent{
			AlertID:    newID(),
			Type:       reason,
			Severity:   "warning",
			TenantID:   tenant,
			CustomerID: customer,
			Message:    "request blocked: " + reason,
			OccurredAt: time.Now().UTC(),
		}
		if err := s.alerter.Publish(ctx, e); err != nil {
			s.log.Error("alert publish failed", "err", err)
		}
	}()
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

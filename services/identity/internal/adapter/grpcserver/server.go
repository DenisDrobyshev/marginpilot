// Package grpcserver adapts the key store to the IdentityService gRPC contract.
package grpcserver

import (
	"context"

	identityv1 "github.com/marginpilot/contracts/gen/identity/v1"

	"github.com/marginpilot/identity/internal/adapter/pgstore"
)

// Server implements identityv1.IdentityServiceServer.
type Server struct {
	identityv1.UnimplementedIdentityServiceServer
	store *pgstore.Store
}

// New wraps a key store as a gRPC server.
func New(store *pgstore.Store) *Server { return &Server{store: store} }

// Resolve maps an opaque API key to its tenant/customer.
func (s *Server) Resolve(ctx context.Context, req *identityv1.ResolveRequest) (*identityv1.ResolveResponse, error) {
	r, found, err := s.store.Resolve(ctx, req.GetApiKey())
	if err != nil {
		return nil, err
	}
	if !found {
		return &identityv1.ResolveResponse{Found: false}, nil
	}
	return &identityv1.ResolveResponse{
		Found:      true,
		TenantId:   r.TenantID,
		CustomerId: r.CustomerID,
		Feature:    r.Feature,
	}, nil
}

// Package resolver contains outbound adapters implementing app.CallerResolver.
package resolver

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "github.com/marginpilot/contracts/gen/identity/v1"

	"github.com/marginpilot/gateway/internal/app"
)

// Identity resolves virtual keys via the identity service.
type Identity struct {
	conn *grpc.ClientConn
	cli  identityv1.IdentityServiceClient
}

// NewIdentity dials the identity service (target e.g. "identity:9102").
func NewIdentity(target string) (*Identity, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Identity{conn: conn, cli: identityv1.NewIdentityServiceClient(conn)}, nil
}

// Resolve maps an API key to its caller, or returns app.ErrKeyNotFound.
func (r *Identity) Resolve(ctx context.Context, apiKey string) (app.Caller, error) {
	resp, err := r.cli.Resolve(ctx, &identityv1.ResolveRequest{ApiKey: apiKey})
	if err != nil {
		return app.Caller{}, err
	}
	if !resp.GetFound() {
		return app.Caller{}, app.ErrKeyNotFound
	}
	return app.Caller{
		TenantID:   resp.GetTenantId(),
		CustomerID: resp.GetCustomerId(),
		Feature:    resp.GetFeature(),
	}, nil
}

// Close tears down the connection.
func (r *Identity) Close() error { return r.conn.Close() }

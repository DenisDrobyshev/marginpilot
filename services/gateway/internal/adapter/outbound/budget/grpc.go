package budget

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	budgetv1 "github.com/marginpilot/contracts/gen/budget/v1"
)

// GRPC is a BudgetChecker backed by the budget service. The connection is lazy,
// so the gateway starts even if budget is not up yet.
type GRPC struct {
	conn *grpc.ClientConn
	cli  budgetv1.BudgetServiceClient
}

// NewGRPC dials the budget service (target e.g. "budget:9101").
func NewGRPC(target string) (*GRPC, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPC{conn: conn, cli: budgetv1.NewBudgetServiceClient(conn)}, nil
}

// Allow calls the budget service on the hot path.
func (b *GRPC) Allow(ctx context.Context, tenantID, customerID string) (bool, error) {
	resp, err := b.cli.Allow(ctx, &budgetv1.AllowRequest{
		TenantId:   tenantID,
		CustomerId: customerID,
	})
	if err != nil {
		return false, err
	}
	return resp.GetAllowed(), nil
}

// Close tears down the connection.
func (b *GRPC) Close() error { return b.conn.Close() }

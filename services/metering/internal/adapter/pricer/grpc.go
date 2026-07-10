package pricer

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	ratingv1 "github.com/marginpilot/contracts/gen/rating/v1"
)

// GRPC prices via the rating service.
type GRPC struct {
	conn *grpc.ClientConn
	cli  ratingv1.RatingServiceClient
}

// NewGRPC dials the rating service (target e.g. "rating:9103").
func NewGRPC(target string) (*GRPC, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPC{conn: conn, cli: ratingv1.NewRatingServiceClient(conn)}, nil
}

// Cost prices a call through the rating catalog.
func (g *GRPC) Cost(ctx context.Context, model string, in, out int) (int64, error) {
	resp, err := g.cli.Price(ctx, &ratingv1.PriceRequest{
		Model:        model,
		InputTokens:  int64(in),
		OutputTokens: int64(out),
	})
	if err != nil {
		return 0, err
	}
	return resp.GetCostMicros(), nil
}

// Close tears down the connection.
func (g *GRPC) Close() error { return g.conn.Close() }

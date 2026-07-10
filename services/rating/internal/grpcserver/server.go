// Package grpcserver adapts the catalog to the RatingService gRPC contract.
package grpcserver

import (
	"context"

	ratingv1 "github.com/marginpilot/contracts/gen/rating/v1"

	"github.com/marginpilot/rating/internal/catalog"
)

// Server implements ratingv1.RatingServiceServer.
type Server struct {
	ratingv1.UnimplementedRatingServiceServer
	store *catalog.Store
}

// New wraps a catalog as a gRPC server.
func New(store *catalog.Store) *Server { return &Server{store: store} }

// Price prices a call from the catalog.
func (s *Server) Price(ctx context.Context, req *ratingv1.PriceRequest) (*ratingv1.PriceResponse, error) {
	c, err := s.store.Price(ctx, req.GetModel(), req.GetInputTokens(), req.GetOutputTokens())
	if err != nil {
		return nil, err
	}
	return &ratingv1.PriceResponse{CostMicros: c}, nil
}

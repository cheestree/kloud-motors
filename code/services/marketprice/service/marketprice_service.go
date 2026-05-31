package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	marketpricepb "services/marketprice/proto"
	"services/marketprice/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	Repo *repository.Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{Repo: repository.NewRepository(db)}
}

func (s *Service) GetAverageMarketPrice(ctx context.Context, req *marketpricepb.AveragePriceRequest) (*marketpricepb.AveragePriceResponse, error) {
	if strings.TrimSpace(req.Brand) == "" || strings.TrimSpace(req.Model) == "" {
		return nil, status.Error(codes.InvalidArgument, "brand and model are required")
	}
	if req.YearFrom != 0 && req.YearTo != 0 && req.YearFrom > req.YearTo {
		return nil, status.Error(codes.InvalidArgument, "year_from cannot be greater than year_to")
	}

	avgPrice, minPrice, maxPrice, count, err := s.Repo.GetAverageMarketPrice(ctx,
		req.Brand, req.Model, req.YearFrom, req.YearTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get average market price: %w", err)
	}
	if count == 0 {
		return nil, status.Error(codes.NotFound, "no listings found for the requested filters")
	}

	return &marketpricepb.AveragePriceResponse{
		Brand:        req.Brand,
		Model:        req.Model,
		AveragePrice: avgPrice,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		ListingCount: count,
	}, nil
}

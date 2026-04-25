package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"services/search/domain"
	"services/search/proto"
	"services/search/repository"
	"services/search/service"
	"services/shared"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	proto.SearchServiceServer
	service *service.SearchService
}

func (s *server) Search(ctx context.Context, req *proto.SearchRequest) (*proto.SearchResponse, error) {
	var isNew *bool
	if req.IsNew != nil {
		v := req.IsNew.Value
		isNew = &v
	}

	includeSold := false
	if req.IncludeSold != nil {
		includeSold = req.IncludeSold.Value
	}

	result, err := s.service.Search(ctx, domain.SearchParams{
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		MinPrice:     req.MinPrice,
		MaxPrice:     req.MaxPrice,
		MaxMileage:   req.MaxMileage,
		FuelType:     req.FuelType,
		BodyClass:    req.BodyClass,
		DriveType:    req.DriveType,
		Transmission: req.Transmission,
		IsNew:        isNew,
		Page:         req.Page,
		PageSize:     req.PageSize,
		IncludeSold:  includeSold,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	return toSearchResponse(result), nil
}

func toSearchResponse(result *domain.SearchResult) *proto.SearchResponse {
	listings := make([]*shared.ListingSummary, 0, len(result.Listings))
	for _, item := range result.Listings {
		listings = append(listings, toListingSummary(item))
	}

	return &proto.SearchResponse{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		Listings: listings,
	}
}

func toListingSummary(item shared.ListingSummary) *shared.ListingSummary {
	return &shared.ListingSummary{
		Id:           item.Id,
		DealerId:     item.DealerId,
		Make:         item.Make,
		Model:        item.Model,
		Year:         item.Year,
		Price:        item.Price,
		Mileage:      item.Mileage,
		FuelType:     item.FuelType,
		BodyClass:    item.BodyClass,
		DriveType:    item.DriveType,
		Transmission: item.Transmission,
		IsNew:        item.IsNew,
		IsSold:       item.IsSold,
		City:         item.City,
		District:     item.District,
		State:        item.State,
		Country:      item.Country,
		LastSeen:     item.LastSeen,
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		logger.Error("failed to open database", "error", err)
	}
	if err := db.Ping(); err != nil {
		logger.Error("failed to connect to database", "error", err)
	}

	lis, err := net.Listen("tcp", ":50056")
	if err != nil {
		logger.Error("error on listen", "error", err)
	}

	repo := repository.NewSearchRepository(db)
	svc := service.NewSearchService(repo)

	s := grpc.NewServer()
	proto.RegisterSearchServiceServer(s, &server{service: svc})

	logger.Info("gRPC server is running on " + lis.Addr().String() + "...")

	if err := s.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}

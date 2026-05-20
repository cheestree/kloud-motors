package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	models "services/listing/models"
	listingpb "services/listing/proto"
	"services/listing/repository"
	"services/listing/service"
	"services/observability"
	"services/redis/cache"
	"services/shared"
	"services/utils"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	listingpb.ListingServiceServer
	service *service.ListingService
}

func (s *server) GetListingDetails(ctx context.Context, req *listingpb.ListingDetailsRequest) (*listingpb.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	listing, err := s.service.GetListingDetails(ctx, req.Id)
	if err != nil {
		return nil, models.MapListingError("fetch listing", err)
	}
	if listing == nil {
		return nil, status.Error(codes.NotFound, "listing not found")
	}

	return models.ToListingDetailsResponse(listing), nil
}

func (s *server) CompareListings(ctx context.Context, req *listingpb.CompareListingsRequest) (*listingpb.CompareListingsResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one listing id is required")
	}

	listings, err := s.service.CompareListings(ctx, req.Ids)
	if err != nil {
		return nil, models.MapListingError("compare listings", err)
	}

	responses := make([]*listingpb.ListingDetailsResponse, 0, len(listings))
	for _, listing := range listings {
		responses = append(responses, models.ToListingDetailsResponse(listing))
	}

	return &listingpb.CompareListingsResponse{Listings: responses}, nil
}

func (s *server) CreateListing(ctx context.Context, req *listingpb.CreateListingRequest) (*listingpb.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing details are required")
	}

	listing, err := s.service.CreateListing(ctx, repository.ListingMutation{
		Vin:          req.Vin,
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		Price:        req.Price,
		Mileage:      req.Mileage,
		City:         req.City,
		District:     req.District,
		State:        req.State,
		Country:      req.Country,
		FuelType:     req.FuelType,
		BodyClass:    req.BodyClass,
		DriveType:    req.DriveType,
		Transmission: req.Transmission,
		Trim:         req.Trim,
		Color:        req.Color,
		DealerID:     req.DealerId,
		IsNew:        req.IsNew,
		IsSold:       req.IsSold,
	})
	if err != nil {
		return nil, models.MapListingError("create listing", err)
	}

	return models.ToListingDetailsResponse(listing), nil
}

func (s *server) UpdateListing(ctx context.Context, req *listingpb.UpdateListingRequest) (*listingpb.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing details are required")
	}

	listing, err := s.service.UpdateListing(ctx, req.Id, repository.ListingMutation{
		Vin:          req.Vin,
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		Price:        req.Price,
		Mileage:      req.Mileage,
		City:         req.City,
		District:     req.District,
		State:        req.State,
		Country:      req.Country,
		FuelType:     req.FuelType,
		BodyClass:    req.BodyClass,
		DriveType:    req.DriveType,
		Transmission: req.Transmission,
		Trim:         req.Trim,
		Color:        req.Color,
		DealerID:     req.DealerId,
		IsNew:        req.IsNew,
	})
	if err != nil {
		return nil, models.MapListingError("update listing", err)
	}

	return models.ToListingDetailsResponse(listing), nil
}

func (s *server) SetListingSoldStatus(ctx context.Context, req *listingpb.SetListingSoldStatusRequest) (*listingpb.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id, dealer id, and sold status are required")
	}
	listing, err := s.service.SetListingSoldStatus(ctx, req.Id, req.DealerId, req.IsSold)
	if err != nil {
		return nil, models.MapListingError("set listing sold status", err)
	}
	return models.ToListingDetailsResponse(listing), nil
}

func (s *server) DeleteListing(ctx context.Context, req *listingpb.DeleteListingRequest) (*listingpb.DeleteListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id and dealer id are required")
	}

	deleted, err := s.service.DeleteListing(ctx, req.Id, req.DealerId)
	if err != nil {
		return nil, models.MapListingError("delete listing", err)
	}

	return &listingpb.DeleteListingResponse{Deleted: deleted}, nil
}

func (s *server) CheckListingOwnership(ctx context.Context, req *listingpb.CheckListingOwnershipRequest) (*listingpb.CheckListingOwnershipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing_id and dealer_id are required")
	}
	isOwner, err := s.service.CheckListingOwnership(ctx, req.ListingId, req.DealerId)
	if err != nil {
		return nil, models.MapListingError("check listing ownership", err)
	}
	return &listingpb.CheckListingOwnershipResponse{IsOwner: isOwner}, nil
}

func (s *server) GetListingSummary(ctx context.Context, req *listingpb.ListingDetailsRequest) (*shared.ListingSummary, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	summary, err := s.service.GetListingSummary(ctx, req.Id)
	if err != nil {
		return nil, models.MapListingError("fetch listing summary", err)
	}
	if summary == nil {
		return nil, status.Error(codes.NotFound, "listing not found")
	}

	return models.ToListingSummary(summary), nil
}

func (s *server) GetListingSummaries(ctx context.Context, req *listingpb.ListingSummariesRequest) (*listingpb.ListingSummariesResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one listing id is required")
	}

	listings, err := s.service.GetListingSummaries(ctx, req.Ids)
	if err != nil {
		return nil, models.MapListingError("fetch listing summaries", err)
	}

	responseListings := make([]*shared.ListingSummary, 0, len(listings))
	for _, listing := range listings {
		responseListings = append(responseListings, models.ToListingSummary(listing))
	}

	return &listingpb.ListingSummariesResponse{Listings: responseListings}, nil
}

func (s *server) CheckListingOpen(ctx context.Context, req *listingpb.CheckListingOpenRequest) (*listingpb.CheckListingOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	open, dealerID, err := s.service.CheckListingOpen(ctx, req.ListingId)
	if err != nil {
		return nil, models.MapListingError("check listing open", err)
	}

	return &listingpb.CheckListingOpenResponse{IsOpen: open, DealerId: dealerID}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctx := context.Background()
	shutdownTracing := observability.InitTracing(ctx, logger, "listing")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	listingDsn := utils.MustGetEnv("LISTING_DATABASE_URL")

	listingDB := utils.TryConnectDB(listingDsn, 8, 10)

	listingGrpcPort := utils.MustGetEnv("LISTING_GRPC_PORT")

	lis := utils.TryListen(listingGrpcPort)

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	repo := repository.NewListingRepository(listingDB)
	redisHost := utils.GetEnv("REDIS_HOST", "redis-cache")
	redisPort := utils.GetEnv("REDIS_PORT", "6379")
	ttlStr := utils.GetEnv("CACHE_TTL_SECONDS", "3600")
	ttlSeconds, _ := strconv.Atoi(ttlStr)
	redisCache := cache.NewRedisCache(redisHost, redisPort, time.Duration(ttlSeconds)*time.Second)

	listingSvc := service.NewListingService(repo, redisCache)
	listingpb.RegisterListingServiceServer(grpcServer, &server{service: listingSvc})

	utils.HealthCheck("listing.ListingService", grpcServer)

	logger.Info("Listing gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}

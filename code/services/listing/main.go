package main

import (
	"context"
	"log/slog"
	"os"

	models "services/listing/models"
	"services/listing/proto"
	"services/listing/repository"
	"services/listing/service"
	"services/shared"
	"services/utils"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	proto.ListingServiceServer
	service *service.ListingService
}

func (s *server) GetListingDetails(ctx context.Context, req *proto.ListingDetailsRequest) (*proto.ListingDetailsResponse, error) {
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

func (s *server) CompareListings(ctx context.Context, req *proto.CompareListingsRequest) (*proto.CompareListingsResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one listing id is required")
	}

	listings, err := s.service.CompareListings(ctx, req.Ids)
	if err != nil {
		return nil, models.MapListingError("compare listings", err)
	}

	responses := make([]*proto.ListingDetailsResponse, 0, len(listings))
	for _, listing := range listings {
		responses = append(responses, models.ToListingDetailsResponse(listing))
	}

	return &proto.CompareListingsResponse{Listings: responses}, nil
}

func (s *server) CreateListing(ctx context.Context, req *proto.CreateListingRequest) (*proto.ListingDetailsResponse, error) {
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

func (s *server) UpdateListing(ctx context.Context, req *proto.UpdateListingRequest) (*proto.ListingDetailsResponse, error) {
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

func (s *server) SetListingSoldStatus(ctx context.Context, req *proto.SetListingSoldStatusRequest) (*proto.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id, dealer id, and sold status are required")
	}
	listing, err := s.service.SetListingSoldStatus(ctx, req.Id, req.DealerId, req.IsSold)
	if err != nil {
		return nil, models.MapListingError("set listing sold status", err)
	}
	return models.ToListingDetailsResponse(listing), nil
}

func (s *server) DeleteListing(ctx context.Context, req *proto.DeleteListingRequest) (*proto.DeleteListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id and dealer id are required")
	}

	deleted, err := s.service.DeleteListing(ctx, req.Id, req.DealerId)
	if err != nil {
		return nil, models.MapListingError("delete listing", err)
	}

	return &proto.DeleteListingResponse{Deleted: deleted}, nil
}

func (s *server) CheckListingOwnership(ctx context.Context, req *proto.CheckListingOwnershipRequest) (*proto.CheckListingOwnershipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing_id and dealer_id are required")
	}
	isOwner, err := s.service.CheckListingOwnership(ctx, req.ListingId, req.DealerId)
	if err != nil {
		return nil, models.MapListingError("check listing ownership", err)
	}
	return &proto.CheckListingOwnershipResponse{IsOwner: isOwner}, nil
}

func (s *server) GetListingSummary(ctx context.Context, req *proto.ListingDetailsRequest) (*shared.ListingSummary, error) {
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

func (s *server) GetListingSummaries(ctx context.Context, req *proto.ListingSummariesRequest) (*proto.ListingSummariesResponse, error) {
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

	return &proto.ListingSummariesResponse{Listings: responseListings}, nil
}

func (s *server) CheckListingOpen(ctx context.Context, req *proto.CheckListingOpenRequest) (*proto.CheckListingOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	open, dealerID, err := s.service.CheckListingOpen(ctx, req.ListingId)
	if err != nil {
		return nil, models.MapListingError("check listing open", err)
	}

	return &proto.CheckListingOpenResponse{IsOpen: open, DealerId: dealerID}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	databaseURL := utils.MustGetEnv("LISTING_DATABASE_URL")

	db := utils.TryConnectDB(databaseURL, 3, 10)

	grpc_port := utils.MustGetEnv("LISTING_GRPC_PORT")

	lis := utils.TryListen(grpc_port)

	repo := repository.NewListingRepository(db)
	svc := service.NewListingService(repo)

	grpcSrv := grpc.NewServer()
	proto.RegisterListingServiceServer(grpcSrv, &server{service: svc})

	logger.Info("Listing gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcSrv, lis)
}

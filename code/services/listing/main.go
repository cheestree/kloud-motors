package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net"
	"os"
	"strings"

	"services/listing/proto"
	"services/listing/repository"
	"services/listing/service"
	"services/shared"

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
		return nil, mapListingError("fetch listing", err)
	}
	if listing == nil {
		return nil, status.Error(codes.NotFound, "listing not found")
	}

	return toListingDetailsResponse(listing), nil
}

func (s *server) CompareListings(ctx context.Context, req *proto.CompareListingsRequest) (*proto.CompareListingsResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one listing id is required")
	}

	listings, err := s.service.CompareListings(ctx, req.Ids)
	if err != nil {
		return nil, mapListingError("compare listings", err)
	}

	responses := make([]*proto.ListingDetailsResponse, 0, len(listings))
	for _, listing := range listings {
		responses = append(responses, toListingDetailsResponse(listing))
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
		return nil, mapListingError("create listing", err)
	}

	return toListingDetailsResponse(listing), nil
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
		return nil, mapListingError("update listing", err)
	}

	return toListingDetailsResponse(listing), nil
}

func (s *server) SetListingSoldStatus(ctx context.Context, req *proto.SetListingSoldStatusRequest) (*proto.ListingDetailsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id, dealer id, and sold status are required")
	}
	listing, err := s.service.SetListingSoldStatus(ctx, req.Id, req.DealerId, req.IsSold)
	if err != nil {
		return nil, mapListingError("set listing sold status", err)
	}
	return toListingDetailsResponse(listing), nil
}

func (s *server) DeleteListing(ctx context.Context, req *proto.DeleteListingRequest) (*proto.DeleteListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id and dealer id are required")
	}

	deleted, err := s.service.DeleteListing(ctx, req.Id, req.DealerId)
	if err != nil {
		return nil, mapListingError("delete listing", err)
	}

	return &proto.DeleteListingResponse{Deleted: deleted}, nil
}

func (s *server) CheckListingOwnership(ctx context.Context, req *proto.CheckListingOwnershipRequest) (*proto.CheckListingOwnershipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing_id and dealer_id are required")
	}
	isOwner, err := s.service.CheckListingOwnership(ctx, req.ListingId, req.DealerId)
	if err != nil {
		return nil, mapListingError("check listing ownership", err)
	}
	return &proto.CheckListingOwnershipResponse{IsOwner: isOwner}, nil
}

func (s *server) GetListingSummary(ctx context.Context, req *proto.ListingDetailsRequest) (*shared.ListingSummary, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	summary, err := s.service.GetListingSummary(ctx, req.Id)
	if err != nil {
		return nil, mapListingError("fetch listing summary", err)
	}
	if summary == nil {
		return nil, status.Error(codes.NotFound, "listing not found")
	}

	return toListingSummary(summary), nil
}

func (s *server) GetListingSummaries(ctx context.Context, req *proto.ListingSummariesRequest) (*proto.ListingSummariesResponse, error) {
	if req == nil || len(req.Ids) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one listing id is required")
	}

	listings, err := s.service.GetListingSummaries(ctx, req.Ids)
	if err != nil {
		return nil, mapListingError("fetch listing summaries", err)
	}

	responseListings := make([]*shared.ListingSummary, 0, len(listings))
	for _, listing := range listings {
		responseListings = append(responseListings, toListingSummary(listing))
	}

	return &proto.ListingSummariesResponse{Listings: responseListings}, nil
}

func toListingSummary(summary *shared.ListingSummary) *shared.ListingSummary {
	if summary == nil {
		return nil
	}

	return &shared.ListingSummary{
		Id:           summary.Id,
		DealerId:     summary.DealerId,
		Make:         summary.Make,
		Model:        summary.Model,
		Year:         summary.Year,
		Price:        summary.Price,
		Mileage:      summary.Mileage,
		FuelType:     summary.FuelType,
		BodyClass:    summary.BodyClass,
		DriveType:    summary.DriveType,
		Transmission: summary.Transmission,
		IsNew:        summary.IsNew,
		City:         summary.City,
		District:     summary.District,
		State:        summary.State,
		Country:      summary.Country,
		LastSeen:     summary.LastSeen,
		IsSold:       summary.IsSold,
	}
}

func (s *server) CheckListingOpen(ctx context.Context, req *proto.CheckListingOpenRequest) (*proto.CheckListingOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	open, dealerID, err := s.service.CheckListingOpen(ctx, req.ListingId)
	if err != nil {
		return nil, mapListingError("check listing open", err)
	}

	return &proto.CheckListingOpenResponse{IsOpen: open, DealerId: dealerID}, nil
}

func (s *server) CheckListingOpen(ctx context.Context, req *proto.CheckListingOpenRequest) (*proto.CheckListingOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing id is required")
	}

	open, err := s.service.CheckListingOpen(ctx, req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check listing open: %v", err)
	}

	return &proto.CheckListingOpenResponse{IsOpen: open}, nil
}

func toListingDetailsResponse(listing *shared.ListingDetails) *proto.ListingDetailsResponse {
	if listing == nil {
		return nil
	}
	return &proto.ListingDetailsResponse{
		Id:           listing.Id,
		Make:         listing.Make,
		Model:        listing.Model,
		Year:         listing.Year,
		Price:        float64(listing.Price),
		Mileage:      listing.Mileage,
		City:         listing.City,
		District:     listing.District,
		State:        listing.State,
		Country:      listing.Country,
		FuelType:     listing.FuelType,
		Trim:         listing.Trim,
		Transmission: listing.Transmission,
		Color:        listing.Color,
		SellerType:   listing.SellerType,
		Description:  listing.Description,
		ListedAt:     listing.LastSeen,
		Images:       listing.Images,
		IsSold:       listing.IsSold,
	}
}

func mapListingError(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, service.ErrListingNotFound) {
		return status.Error(codes.NotFound, "listing not found")
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "required") ||
		strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "must be") ||
		strings.Contains(lower, "cannot") ||
		strings.Contains(lower, "unknown") {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique") {
		return status.Error(codes.AlreadyExists, "listing with this VIN already exists")
	}

	return status.Errorf(codes.Internal, "failed to %s: %v", action, err)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	lis, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	repo := repository.NewListingRepository(db)
	svc := service.NewListingService(repo)

	s := grpc.NewServer()
	proto.RegisterListingServiceServer(s, &server{service: svc})

	log.Println("Listing gRPC server is running on " + lis.Addr().String() + "...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

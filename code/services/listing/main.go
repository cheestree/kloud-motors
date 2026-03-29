package main

import (
	"context"
	"database/sql"
	"listing/domain"
	proto "listing/proto"
	"listing/repository"
	"listing/service"
	"log"
	"net"
	"os"

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
		return nil, status.Errorf(codes.Internal, "failed to fetch listing: %v", err)
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
		return nil, status.Errorf(codes.Internal, "failed to compare listings: %v", err)
	}

	responses := make([]*proto.ListingDetailsResponse, 0, len(listings))
	for _, listing := range listings {
		responses = append(responses, toListingDetailsResponse(listing))
	}

	return &proto.CompareListingsResponse{Listings: responses}, nil
}

/*
func (s *server) CreateListing(ctx context.Context, req *proto.CreateListingRequest) (*proto.ListingDetailsResponse, error) {
	return &proto.ListingDetailsResponse{}, nil
}

func (s *server) UpdateListing(ctx context.Context, req *proto.UpdateListingRequest) (*proto.ListingDetailsResponse, error) {
	return &proto.ListingDetailsResponse{}, nil
}

func (s *server) DeleteListing(ctx context.Context, req *proto.DeleteListingRequest) (*proto.DeleteListingResponse, error) {
	return &proto.DeleteListingResponse{Success: true}, nil
}
*/

func (s *server) CheckListingOpen(ctx context.Context, req *proto.CheckListingOpenRequest) (*proto.CheckListingOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing_id is required")
	}
	isOpen, err := s.service.CheckListingOpen(ctx, req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check listing open: %v", err)
	}
	return &proto.CheckListingOpenResponse{IsOpen: isOpen}, nil
}

func (s *server) CheckListingOwnership(ctx context.Context, req *proto.CheckListingOwnershipRequest) (*proto.CheckListingOwnershipResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing_id and dealer_id are required")
	}
	isOwner, err := s.service.CheckListingOwnership(ctx, req.ListingId, req.DealerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check listing ownership: %v", err)
	}
	return &proto.CheckListingOwnershipResponse{IsOwner: isOwner}, nil
}

func toListingDetailsResponse(listing *domain.ListingDetails) *proto.ListingDetailsResponse {
	if listing == nil {
		return nil
	}
	return &proto.ListingDetailsResponse{
		Id:           listing.ID,
		Make:         listing.Make,
		Model:        listing.Model,
		Year:         listing.Year,
		Price:        listing.Price,
		Mileage:      listing.Mileage,
		Location:     listing.Location,
		FuelType:     listing.FuelType,
		Trim:         listing.Trim,
		Transmission: listing.Transmission,
		Color:        listing.Color,
		SellerType:   listing.SellerType,
		Description:  listing.Description,
		ListedAt:     listing.ListedAt,
		Images:       listing.Images,
	}
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

	lis, err := net.Listen("tcp", ":50052")
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

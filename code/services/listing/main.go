package main

import (
	"context"
	proto "listing/proto"
	"log"
	"net"

	"google.golang.org/grpc"
)

type server struct {
	proto.ListingServiceServer
}

func (s *server) GetListingDetails(ctx context.Context, req *proto.ListingDetailsRequest) (*proto.ListingDetailsResponse, error) {
	return &proto.ListingDetailsResponse{}, nil
}

func (s *server) CompareListings(ctx context.Context, req *proto.CompareListingsRequest) (*proto.CompareListingsResponse, error) {
	return &proto.CompareListingsResponse{}, nil
}

func (s *server) CreateListing(ctx context.Context, req *proto.CreateListingRequest) (*proto.ListingDetailsResponse, error) {
	return &proto.ListingDetailsResponse{}, nil
}

func (s *server) UpdateListing(ctx context.Context, req *proto.UpdateListingRequest) (*proto.ListingDetailsResponse, error) {
	return &proto.ListingDetailsResponse{}, nil
}

func (s *server) DeleteListing(ctx context.Context, req *proto.DeleteListingRequest) (*proto.DeleteListingResponse, error) {
	return &proto.DeleteListingResponse{Success: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterListingServiceServer(s, &server{})

	log.Println("Listing gRPC server is running on " + lis.Addr().String() + "...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

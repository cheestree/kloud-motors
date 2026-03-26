package main

import (
	"context"
	"log"
	"net"
	proto "search/proto"

	"google.golang.org/grpc"
)

type server struct {
	proto.SearchServiceServer
}

func (s *server) Search(ctx context.Context, req *proto.SearchRequest) (*proto.SearchResponse, error) {
	return &proto.SearchResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterSearchServiceServer(s, &server{})

	log.Println("gRPC server is running on " + lis.Addr().String() + "...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

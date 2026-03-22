package main

import (
	"context"
	"log"
	"net"

	proto "chat/proto"

	"google.golang.org/grpc"
)

type server struct {
	proto.ChatServiceServer
}

func (s *server) OpenChat(ctx context.Context, req *proto.OpenChatRequest) (*proto.OpenChatResponse, error) {
	// verify if the two users exist or not and the seller is really a seller
	// verify if the listing belongs to the seller
	// create or return a new chat between the buyer and the seller
	return &proto.OpenChatResponse{}, nil
}

func (s *server) GetChatHistory(ctx context.Context, req *proto.GetChatHistoryRequest) (*proto.GetChatHistoryResponse, error) {
	// verify if the chat exists
	// verify if the listing is still open
	// verify if the requesting user belongs to the chat
	return &proto.GetChatHistoryResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterChatServiceServer(s, &server{})

	log.Println("gRPC server is running on " + lis.Addr().String() + "...")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

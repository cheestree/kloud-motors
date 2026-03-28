package main

import (
	proto "chat/proto"
	"context"
)

type grpcServer struct {
	proto.ChatServiceServer
	//db
}

func (s *grpcServer) OpenChat(ctx context.Context, req *proto.OpenChatRequest) (*proto.OpenChatResponse, error) {
	// verify if the two users exist or not and the seller is really a seller
	// verify if the listing belongs to the seller
	// create or return a new chat between the buyer and the seller
	return &proto.OpenChatResponse{}, nil
}

func (s *grpcServer) GetChatHistory(ctx context.Context, req *proto.GetChatHistoryRequest) (*proto.GetChatHistoryResponse, error) {
	// verify if the chat exists
	// verify if the listing is still open
	// verify if the requesting user belongs to the chat
	return &proto.GetChatHistoryResponse{}, nil
}

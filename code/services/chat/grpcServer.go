package main

import (
	proto "chat/proto"
	"chat/repository"
	"context"
	"crypto/sha1"
	"encoding/hex"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	proto.ChatServiceServer
	messageStore repository.MessageRepo
	indexStore   repository.ChatIndexRepo
	historyLimit int
}

func (s *grpcServer) GetActiveChats(ctx context.Context, req *proto.GetActiveChatsRequest) (*proto.GetActiveChatsResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	if s.indexStore == nil {
		return &proto.GetActiveChatsResponse{Chats: []*proto.ChatsSummary{}}, nil
	}

	chats, err := s.indexStore.ListUserChats(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list active chats: %v", err)
	}

	protoChats := make([]*proto.ChatsSummary, 0, len(chats))
	for _, chat := range chats {
		protoChats = append(protoChats, &proto.ChatsSummary{
			ChatId:    chat.ChatID,
			ListingId: chat.ListingID,
			Brand:     chat.Brand,
			Model:     chat.Model,
		})
	}

	return &proto.GetActiveChatsResponse{Chats: protoChats}, nil
}

func (s *grpcServer) OpenChat(ctx context.Context, req *proto.OpenChatRequest) (*proto.OpenChatResponse, error) {
	if req.GetUserId() == "" || req.GetSellerId() == "" || req.GetListingId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, seller_id and listing_id are required")
	}

	listingID := req.GetListingId()
	brand := "Unknown" // TODO: Get from listing service
	model := "Unknown" // TODO: Get from listing service

	var chatID string
	if s.indexStore != nil {
		var err error
		chatID, err = s.indexStore.UpsertChatParticipant(ctx, req.GetUserId(), listingID, brand, model)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to index chat participant: %v", err)
		}

		_, err = s.indexStore.UpsertChatParticipant(ctx, req.GetSellerId(), listingID, brand, model)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to index seller participant: %v", err)
		}
	} else {
		// TODO: Check if it is the correct way to do
		chatID = buildChatID(listingID, req.GetUserId(), req.GetSellerId())
	}

	return &proto.OpenChatResponse{ChatId: chatID}, nil
}

func (s *grpcServer) GetChatHistory(ctx context.Context, req *proto.GetChatHistoryRequest) (*proto.GetChatHistoryResponse, error) {
	if req.GetChatId() == "" || req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "chat_id and user_id are required")
	}

	if s.indexStore != nil {
		canAccess, err := s.indexStore.UserCanAccessChat(ctx, req.GetUserId(), req.GetChatId())
		if err != nil {
			return nil, err
		}

		if !canAccess {
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		}
	}

	if s.messageStore == nil {
		return &proto.GetChatHistoryResponse{Messages: []*proto.ChatMessage{}}, nil
	}

	limit := s.historyLimit
	if limit <= 0 {
		limit = 50
	}

	messages, err := s.messageStore.ListChatMessages(ctx, req.GetChatId(), limit, 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load chat history: %v", err)
	}

	protoMessages := make([]*proto.ChatMessage, 0, len(messages))
	for _, message := range messages {
		protoMessages = append(protoMessages, &proto.ChatMessage{
			SenderId:  message.UserID,
			Content:   message.Message,
			Timestamp: message.Time.UnixMilli(),
		})
	}

	return &proto.GetChatHistoryResponse{Messages: protoMessages}, nil
}

func buildChatID(listingID, userID, sellerID string) string {
	h := sha1.Sum([]byte(listingID + ":" + userID + ":" + sellerID))
	return "chat_" + hex.EncodeToString(h[:10])
}

package main

import (
	proto "services/chat/proto"
	"services/chat/repository"
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	proto.ChatServiceServer
	messageStore repository.MessageRepo
	indexStore   repository.ChatIndexRepo
	historyLimit int32

	listingClient proto.ListingServiceClient
	sellerClient  proto.SellerServiceClient
}

func (s *grpcServer) GetActiveChats(ctx context.Context, req *proto.GetActiveChatsRequest) (*proto.GetActiveChatsResponse, error) {
	if req.GetUserId() < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
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
	if req.GetUserId() < 0 || req.GetSellerId() < 0 || req.GetListingId() < 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, seller_id and listing_id are required")
	}

	sellerId := req.GetSellerId()
	isSeller, err := s.sellerClient.VerifySellerProfile(ctx, &proto.VerifySellerRequest{SellerId: sellerId})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify seller profile: %v", err)
	}

	if !isSeller.IsSeller {
		return nil, status.Error(codes.InvalidArgument, "seller not allowed")
	}

	listingID := req.GetListingId()
	isListingFromSeller, err := s.listingClient.CheckListingOwnership(ctx,
		&proto.CheckListingOwnershipRequest{ListingId: listingID, DealerId: sellerId})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check listing ownership: %v", err)
	}
	if !isListingFromSeller.IsOwner {
		return nil, status.Error(codes.InvalidArgument, "seller not allowed")
	}

	listing, err := s.listingClient.GetListingSummary(ctx, &proto.ListingDetailsRequest{Id: listingID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get listing details: %v", err)
	}

	brand := listing.Maker
	model := listing.Model

	var chatID string
	if s.indexStore != nil {
		var err error
		chatID, err = s.indexStore.UpsertChatParticipant(ctx, req.GetUserId(), req.GetSellerId(), listingID, brand, model)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to index chat participants: %v", err)
		}

		return &proto.OpenChatResponse{ChatId: chatID}, nil
	} else {
		return nil, status.Errorf(codes.Internal, "failed to index chat participants")
	}
}

func (s *grpcServer) GetChatHistory(ctx context.Context, req *proto.GetChatHistoryRequest) (*proto.GetChatHistoryResponse, error) {
	if req.GetChatId() == "" || req.GetUserId() < 0 {
		return nil, status.Error(codes.InvalidArgument, "chat_id and user_id are required and be a valid value")
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

	if req.GetLimit() <= 0 || req.GetLimit() > s.historyLimit {
		return nil, status.Error(codes.InvalidArgument, "invalid limit")
	}

	if req.GetSkip() < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid limit")
	}

	messages, err := s.messageStore.ListChatMessages(ctx, req.GetChatId(), req.GetLimit(), req.GetSkip())
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

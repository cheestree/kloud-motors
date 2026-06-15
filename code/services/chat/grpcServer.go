package main

import (
	"context"

	proto "services/chat/proto"
	"services/chat/repository"
	listingproto "services/listing/proto"
	sellerproto "services/seller/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	proto.ChatServiceServer
	messageStore repository.MessageRepo
	indexStore   repository.ChatIndexRepo
	historyLimit int32

	listingClient listingproto.ListingServiceClient
	sellerClient  sellerproto.SellerServiceClient
}

const defaultChatHistoryLimit int32 = 20

func (s *grpcServer) GetChats(ctx context.Context, req *proto.GetChatsRequest) (*proto.GetChatsResponse, error) {
	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if s.indexStore == nil {
		return &proto.GetChatsResponse{Chats: []*proto.ChatsSummary{}}, nil
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

	return &proto.GetChatsResponse{Chats: protoChats}, nil
}

func (s *grpcServer) OpenChat(ctx context.Context, req *proto.OpenChatRequest) (*proto.OpenChatResponse, error) {
	if req.GetUserId() <= 0 || req.GetSellerId() <= 0 || req.GetListingId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id, seller_id and listing_id are required")
	}

	sellerId := req.GetSellerId()
	isSeller, err := s.sellerClient.VerifySellerProfile(ctx, &sellerproto.VerifySellerRequest{SellerId: sellerId})

	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, status.Errorf(codes.Unavailable, "seller service unavailable: %v", err)
		case codes.NotFound:
			return nil, status.Error(codes.NotFound, "seller not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify seller profile: %v", err)
	}

	if !isSeller.IsSeller {
		return nil, status.Error(codes.NotFound, "seller not found")
	}

	listingID := req.GetListingId()
	listing, err := s.listingClient.GetListingSummary(ctx, &listingproto.ListingDetailsRequest{Id: listingID})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, status.Errorf(codes.Unavailable, "listing service unavailable: %v", err)
		case codes.NotFound:
			return nil, status.Error(codes.NotFound, "listing not found")
		case codes.InvalidArgument:
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to get listing details: %v", err)
	}

	isListingFromSeller, err := s.listingClient.CheckListingOwnership(ctx,
		&listingproto.CheckListingOwnershipRequest{ListingId: listingID, SellerId: sellerId})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, status.Errorf(codes.Unavailable, "listing service unavailable: %v", err)
		case codes.NotFound:
			return nil, status.Error(codes.NotFound, "listing not found")
		case codes.InvalidArgument:
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to check listing ownership: %v", err)
	}
	if !isListingFromSeller.IsOwner {
		return nil, status.Error(codes.PermissionDenied, "listing does not belong to this seller")
	}

	brand := listing.Make
	model := listing.Model
	isSold := listing.IsSold

	if isSold {
		return nil, status.Error(codes.FailedPrecondition, "listing is already sold")
	}

	existingChats, err := s.indexStore.GetChatsFromListingSeller(ctx, listingID, req.GetUserId())
	if err == nil && len(existingChats) > 0 {
		return &proto.OpenChatResponse{ChatId: existingChats[0]}, nil
	}

	var chatID string
	if s.indexStore != nil {
		var err error
		chatID, err = s.indexStore.UpsertChatParticipant(ctx, req.GetUserId(), req.GetSellerId(), listingID, brand, model)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to index chat participants: %v", err)
		}
		return &proto.OpenChatResponse{ChatId: chatID, IsChatClosed: isSold}, nil
	}

	return nil, status.Errorf(codes.Internal, "failed to index chat participants")
}

func (s *grpcServer) GetChatHistory(ctx context.Context, req *proto.GetChatHistoryRequest) (*proto.GetChatHistoryResponse, error) {
	if req.GetChatId() == "" || req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chat_id and user_id are required and be a valid value")
	}

	if s.indexStore != nil {
		canAccess, err := s.indexStore.UserCanAccessChat(ctx, req.GetUserId(), req.GetChatId())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check chat access: %v", err)
		}

		if !canAccess {
			return nil, status.Error(codes.PermissionDenied, "permission denied")
		}
	}

	limit := req.GetLimit()
	if limit <= 0 {
		limit = defaultChatHistoryLimit
	}
	if s.historyLimit > 0 && limit > s.historyLimit {
		limit = s.historyLimit // Cap at max allowed
	}

	skip := req.GetSkip()
	if skip < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid skip")
	}

	if s.messageStore == nil {
		return &proto.GetChatHistoryResponse{
			Messages:   []*proto.ChatMessage{},
			Pagination: chatPagination(limit, skip, false),
		}, nil
	}

	messages, err := s.messageStore.ListChatMessages(ctx, req.GetChatId(), limit+1, skip)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load chat history: %v", err)
	}

	hasNext := int32(len(messages)) > limit
	if hasNext {
		messages = messages[:limit]
	}

	protoMessages := make([]*proto.ChatMessage, 0, len(messages))
	for _, message := range messages {
		protoMessages = append(protoMessages, &proto.ChatMessage{
			SenderId:  message.UserID,
			Content:   message.Message,
			Timestamp: message.Time.UnixMilli(),
		})
	}

	return &proto.GetChatHistoryResponse{
		Messages:   protoMessages,
		Pagination: chatPagination(limit, skip, hasNext),
	}, nil
}

func chatPagination(limit, skip int32, hasNext bool) *proto.Pagination {
	pagination := &proto.Pagination{
		Limit:   limit,
		Skip:    skip,
		HasNext: hasNext,
	}
	if hasNext {
		nextSkip := skip + limit
		pagination.NextSkip = &nextSkip
	}
	return pagination
}

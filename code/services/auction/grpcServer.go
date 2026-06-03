package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	proto "services/auction/proto"
	ws2 "services/auction/ws"
	listingproto "services/listing/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	proto.AuctionServiceServer
	hub           *ws2.Hub
	listingClient listingproto.ListingServiceClient
}

const (
	defaultAuctionPageSize = 10
	maxAuctionPageSize     = 100
)

func normalizePage(page, limit int32) (int, int) {
	normalizedLimit := int(limit)
	if normalizedLimit <= 0 {
		normalizedLimit = defaultAuctionPageSize
	}
	if normalizedLimit > maxAuctionPageSize {
		normalizedLimit = maxAuctionPageSize
	}

	normalizedPage := int(page)
	if normalizedPage <= 0 {
		normalizedPage = 1
	}

	return normalizedPage, normalizedLimit
}

func (s *server) ListAuctions(ctx context.Context, req *proto.ListAuctionsRequest) (*proto.ListAuctionsResponse, error) {
	query := `SELECT id, listing_id, seller_id, starting_price, current_price, status, end_time, winner_user_id, created_at, reserve_met, total_bids FROM auctions WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM auctions WHERE 1=1`

	var args []interface{}
	argId := 1

	if req.Status != "" {
		filter := fmt.Sprintf(` AND status = $%d`, argId)
		query += filter
		countQuery += filter
		args = append(args, req.Status)
		argId++
	}

	page, limit := normalizePage(req.Page, req.Limit)
	offset := (page - 1) * limit

	var total int32
	err := auctionDB.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Erro em ListAuctions (Count): %v", err)
		return nil, status.Error(codes.Internal, "failed to count auctions")
	}

	query += fmt.Sprintf(` ORDER BY created_at DESC, id ASC LIMIT $%d OFFSET $%d`, argId, argId+1)
	args = append(args, limit, offset)

	rows, err := auctionDB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Erro em ListAuctions (Query): %v", err)
		return nil, status.Error(codes.Internal, "failed to list auctions")
	}
	defer rows.Close()

	var protoAuctions []*proto.Auction
	for rows.Next() {
		var (
			id, status          string
			listingId, sellerId int64
			startingPrice       float64
			currentPrice        sql.NullFloat64
			endTime             time.Time
			winnerUserId        sql.NullString
			createdAt           time.Time
			reserveMet          bool
			totalBids           int32
		)

		if err := rows.Scan(&id, &listingId, &sellerId, &startingPrice, &currentPrice, &status, &endTime, &winnerUserId, &createdAt, &reserveMet, &totalBids); err != nil {
			log.Printf("Erro ao ler auction: %v", err)
			continue
		}

		var currentPricePtr *float64
		if currentPrice.Valid {
			val := currentPrice.Float64
			currentPricePtr = &val
		}

		var winnerUserIdPtr *string
		if winnerUserId.Valid {
			val := winnerUserId.String
			winnerUserIdPtr = &val
		}

		protoAuctions = append(protoAuctions, &proto.Auction{
			AuctionId:     id,
			ListingId:     listingId,
			SellerId:      sellerId,
			StartingPrice: startingPrice,
			CurrentPrice:  currentPricePtr,
			Status:        status,
			EndTime:       endTime.Format(time.RFC3339),
			WinnerUserId:  winnerUserIdPtr,
			CreatedAt:     createdAt.Format(time.RFC3339),
			ReserveMet:    reserveMet,
			TotalBids:     totalBids,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erro no cursor rows (ListAuctions): %v", err)
		return nil, status.Error(codes.Internal, "failed to list auctions")
	}

	return &proto.ListAuctionsResponse{
		Auctions: protoAuctions,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(limit),
	}, nil
}

func (s *server) CreateAuction(ctx context.Context, req *proto.CreateAuctionRequest) (*proto.CreateAuctionResponse, error) {
	if req.ListingId <= 0 || req.SellerId <= 0 || req.StartingPrice <= 0 || req.EndTime == "" {
		return nil, status.Error(codes.InvalidArgument, "listing_id, seller_id, starting_price, and end_time are required")
	}

	openResp, err := s.listingClient.CheckListingOpen(ctx, &listingproto.CheckListingOpenRequest{
		ListingId: req.ListingId,
	})
	if err != nil {
		log.Printf("Error checking listing status: %v", err)
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, status.Error(codes.Unavailable, "listing service unavailable")
		case codes.NotFound, codes.InvalidArgument:
			return nil, err
		}
		return nil, status.Error(codes.Internal, "failed to verify listing status")
	}
	if !openResp.IsOpen {
		return nil, status.Errorf(codes.FailedPrecondition, "listing %v is not available for auction", req.ListingId)
	}

	ownerResp, err := s.listingClient.CheckListingOwnership(ctx, &listingproto.CheckListingOwnershipRequest{
		ListingId: req.ListingId,
		SellerId:  req.SellerId,
	})
	if err != nil {
		log.Printf("Error checking listing ownership: %v", err)
		switch status.Code(err) {
		case codes.Unavailable:
			return nil, status.Error(codes.Unavailable, "listing service unavailable")
		case codes.NotFound, codes.InvalidArgument:
			return nil, err
		}
		return nil, status.Error(codes.Internal, "failed to verify listing ownership")
	}
	if !ownerResp.IsOwner {
		return nil, status.Errorf(codes.PermissionDenied, "seller does not own listing %v", req.ListingId)
	}

	var auctionExists bool
	err = auctionDB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM auctions WHERE listing_id = $1 AND (status = 'ACTIVE' OR status = 'COMPLETED'))", req.ListingId).Scan(&auctionExists)
	if err != nil {
		log.Printf("Error checking for existing auction: %v", err)
		return nil, status.Error(codes.Internal, "failed to check for existing auction")
	}
	if auctionExists {
		return nil, status.Errorf(codes.AlreadyExists, "an active or completed auction already exists for listing %v", req.ListingId)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		log.Printf("Error parsing end_time: %v", err)
		return nil, status.Error(codes.InvalidArgument, "invalid end_time format, expected RFC3339")
	}

	query := `INSERT INTO auctions (listing_id, seller_id, starting_price, reserve_price, status, end_time, created_at, reserve_met, total_bids)
	          VALUES ($1, $2, $3, $4, 'ACTIVE', $5, NOW(), false, 0)
	          RETURNING id, created_at`

	var newId string
	var createdAt time.Time

	err = auctionDB.QueryRowContext(ctx, query,
		req.ListingId,
		req.SellerId,
		req.StartingPrice,
		req.ReservePrice,
		endTime,
	).Scan(&newId, &createdAt)

	if err != nil {
		log.Printf("Error in CreateAuction (Insert): %v", err)
		return nil, status.Error(codes.Internal, "failed to create auction")
	}

	protoAuction := &proto.Auction{
		AuctionId:     newId,
		ListingId:     req.ListingId,
		SellerId:      req.SellerId,
		StartingPrice: req.StartingPrice,
		CurrentPrice:  nil,
		Status:        "ACTIVE",
		EndTime:       req.EndTime,
		WinnerUserId:  nil,
		CreatedAt:     createdAt.Format(time.RFC3339),
		ReserveMet:    false,
		TotalBids:     0,
	}

	return &proto.CreateAuctionResponse{Auction: protoAuction}, nil
}

func (s *server) GetAuctionDetails(ctx context.Context, req *proto.GetAuctionRequest) (*proto.GetAuctionResponse, error) {
	if req.AuctionId == "" {
		return nil, status.Error(codes.InvalidArgument, "auction_id is required")
	}

	query := `SELECT id, listing_id, seller_id, starting_price, current_price, status, end_time, winner_user_id, created_at, reserve_met, total_bids 
	          FROM auctions WHERE id = $1`

	var (
		id, auctionStatus   string
		listingId, sellerId int64
		startingPrice       float64
		currentPrice        sql.NullFloat64
		endTime             time.Time
		winnerUserId        sql.NullString
		createdAt           time.Time
		reserveMet          bool
		totalBids           int32
	)

	err := auctionDB.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &listingId, &sellerId, &startingPrice, &currentPrice,
		&auctionStatus, &endTime, &winnerUserId, &createdAt, &reserveMet, &totalBids,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not supported/found", req.AuctionId)
			return nil, status.Error(codes.NotFound, "auction not found")
		}
		log.Printf("Error in GetAuctionDetails (Query): %v", err)
		return nil, status.Error(codes.Internal, "failed to get auction details")
	}

	var currentPricePtr *float64
	if currentPrice.Valid {
		val := currentPrice.Float64
		currentPricePtr = &val
	}

	var winnerUserIdPtr *string
	if winnerUserId.Valid {
		val := winnerUserId.String
		winnerUserIdPtr = &val
	}

	protoAuction := &proto.Auction{
		AuctionId:     id,
		ListingId:     listingId,
		SellerId:      sellerId,
		StartingPrice: startingPrice,
		CurrentPrice:  currentPricePtr,
		Status:        auctionStatus,
		EndTime:       endTime.Format(time.RFC3339),
		WinnerUserId:  winnerUserIdPtr,
		CreatedAt:     createdAt.Format(time.RFC3339),
		ReserveMet:    reserveMet,
		TotalBids:     totalBids,
	}
	return &proto.GetAuctionResponse{Auction: protoAuction}, nil
}

func (s *server) DeleteAuction(ctx context.Context, req *proto.DeleteAuctionRequest) (*proto.DeleteAuctionResponse, error) {
	if req.AuctionId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "auction_id and user_id are required")
	}

	// 1. Verify if the auction exists
	query := `SELECT id, seller_id FROM auctions WHERE id = $1`

	var (
		id, sellerId string
	)

	err := auctionDB.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &sellerId,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not supported/found", req.AuctionId)
			return nil, status.Error(codes.NotFound, "auction not found")
		}
		log.Printf("Error in DeleteAuction (Query): %v", err)
		return nil, status.Error(codes.Internal, "failed to get auction")
	}

	// 2. Verify if the user deleting is the seller
	if sellerId != req.UserId {
		log.Printf("User %s is not the seller of auction %s", req.UserId, req.AuctionId)
		return nil, status.Error(codes.PermissionDenied, "user not authorized to delete auction")
	}

	// 3. Cancel the auction (soft delete via status)
	_, err = auctionDB.ExecContext(ctx, `UPDATE auctions SET status = 'CANCELLED' WHERE id = $1`, req.AuctionId)
	if err != nil {
		log.Printf("Error in DeleteAuction (Update): %v", err)
		return nil, status.Error(codes.Internal, "failed to delete auction")
	}

	return &proto.DeleteAuctionResponse{Success: true}, nil
}

func (s *server) PlaceBid(ctx context.Context, req *proto.PlaceBidRequest) (*proto.PlaceBidResponse, error) {
	if req.AuctionId == "" || req.BidderId == "" || req.BidAmount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "auction_id, bidder_id, and bid_amount are required")
	}

	// 1. Verify if the auction exists and is ACTIVE
	query := `SELECT id, seller_id, starting_price, current_price, end_time FROM auctions WHERE id = $1 AND status = 'ACTIVE'`

	var (
		id, sellerId  string
		startingPrice float64
		currentPrice  sql.NullFloat64
		endTime       time.Time
	)

	err := auctionDB.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &sellerId, &startingPrice, &currentPrice, &endTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not found or not active", req.AuctionId)
			return nil, status.Error(codes.NotFound, "auction not found or not active")
		}
		log.Printf("Error in PlaceBid (Query): %v", err)
		return nil, status.Error(codes.Internal, "failed to get auction")
	}

	// 2. Bid must be strictly greater than current_price (or starting_price if no bids yet)
	minimumBid := startingPrice
	if currentPrice.Valid {
		minimumBid = currentPrice.Float64
	}
	if req.BidAmount <= minimumBid {
		log.Printf("Bid %f is not greater than minimum %f", req.BidAmount, minimumBid)
		return nil, status.Errorf(codes.InvalidArgument, "bid amount must be greater than current price (%.2f)", minimumBid)
	}

	// 3. Insert the bid and get back the generated id
	var (
		newBidID     string
		bidTimestamp time.Time
	)
	err = auctionDB.QueryRowContext(ctx,
		`INSERT INTO bids (id, auction_id, bidder_id, bid_amount, timestamp)
		 VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		 RETURNING id, timestamp`,
		req.AuctionId, req.BidderId, req.BidAmount,
	).Scan(&newBidID, &bidTimestamp)
	if err != nil {
		log.Printf("Error in PlaceBid (Insert): %v", err)
		return nil, status.Error(codes.Internal, "failed to place bid")
	}

	// 4. Update current_price and total_bids in the auctions table
	_, err = auctionDB.ExecContext(ctx,
		`UPDATE auctions SET current_price = $1, total_bids = total_bids + 1 WHERE id = $2`,
		req.BidAmount, req.AuctionId,
	)
	if err != nil {
		log.Printf("Error in PlaceBid (Update): %v", err)
		return nil, status.Error(codes.Internal, "failed to update auction bid state")
	}

	// 5. Notify all WebSocket clients watching this auction in real-time
	msg := fmt.Sprintf("new bid was placed in %s with the amount: %.2f", req.AuctionId, req.BidAmount)
	if pubErr := s.hub.Publish(req.AuctionId, []byte(msg)); pubErr != nil {
		log.Printf("ws publish error for auction %s: %v", req.AuctionId, pubErr)
	}

	return &proto.PlaceBidResponse{
		Bid: &proto.Bid{
			BidId:     newBidID,
			AuctionId: req.AuctionId,
			BidderId:  req.BidderId,
			BidAmount: req.BidAmount,
			Timestamp: bidTimestamp.UTC().Format(time.RFC3339),
		},
	}, nil
}

func (s *server) GetAuctionBids(ctx context.Context, req *proto.GetAuctionBidsRequest) (*proto.GetAuctionBidsResponse, error) {
	if req.AuctionId == "" {
		return nil, status.Error(codes.InvalidArgument, "auction_id is required")
	}

	var auctionExists bool
	if err := auctionDB.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM auctions WHERE id = $1)", req.AuctionId).Scan(&auctionExists); err != nil {
		log.Printf("Error in GetAuctionBids (Auction Exists): %v", err)
		return nil, status.Error(codes.Internal, "failed to check auction")
	}
	if !auctionExists {
		return nil, status.Error(codes.NotFound, "auction not found")
	}

	// 1. Get total count first
	countQuery := `SELECT COUNT(*) FROM bids WHERE auction_id = $1`
	var total int32
	err := auctionDB.QueryRowContext(ctx, countQuery, req.AuctionId).Scan(&total)
	if err != nil {
		log.Printf("Error in GetAuctionBids (Count): %v", err)
		return nil, status.Error(codes.Internal, "failed to count auction bids")
	}

	page, limit := normalizePage(req.Page, req.Limit)
	offset := (page - 1) * limit

	// 2. Get all bids for a specific auction, ordered by amount descending
	query := `SELECT id, auction_id, bidder_id, bid_amount, timestamp FROM bids WHERE auction_id = $1 ORDER BY bid_amount DESC, timestamp ASC, id ASC LIMIT $2 OFFSET $3`

	rows, err := auctionDB.QueryContext(ctx, query, req.AuctionId, limit, offset)
	if err != nil {
		log.Printf("Error in GetAuctionBids (Query): %v", err)
		return nil, status.Error(codes.Internal, "failed to get auction bids")
	}
	defer rows.Close()

	var protoBids []*proto.Bid
	for rows.Next() {
		var (
			id, auctionId, bidderId string
			bidAmount               float64
			timestamp               time.Time
		)

		if err := rows.Scan(&id, &auctionId, &bidderId, &bidAmount, &timestamp); err != nil {
			log.Printf("Error reading bid: %v", err)
			continue
		}

		protoBids = append(protoBids, &proto.Bid{
			BidId:     id,
			AuctionId: auctionId,
			BidderId:  bidderId,
			BidAmount: bidAmount,
			Timestamp: timestamp.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error in GetAuctionBids cursor: %v", err)
		return nil, status.Error(codes.Internal, "failed to get auction bids")
	}

	return &proto.GetAuctionBidsResponse{
		Bids:     protoBids,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(limit),
	}, nil
}

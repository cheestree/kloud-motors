package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	proto "services/auction/proto"
	ws2 "services/auction/ws"
	listingproto "services/listing/proto"
)

type server struct {
	proto.UnimplementedAuctionServiceServer
	hub           *ws2.Hub
	listingClient listingproto.ListingServiceClient
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

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int32
	err := db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		log.Printf("Erro em ListAuctions (Count): %v", err)
		return nil, err
	}

	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argId, argId+1)
	args = append(args, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		log.Printf("Erro em ListAuctions (Query): %v", err)
		return nil, err
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
		return nil, err
	}

	return &proto.ListAuctionsResponse{
		Auctions: protoAuctions,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(limit),
	}, nil
}

func (s *server) CreateAuction(ctx context.Context, req *proto.CreateAuctionRequest) (*proto.CreateAuctionResponse, error) {

	ownerResp, err := s.listingClient.CheckListingOwnership(ctx, &listingproto.CheckListingOwnershipRequest{
		ListingId: req.ListingId,
		DealerId:  req.SellerId,
	})
	if err != nil {
		log.Printf("Error checking listing ownership: %v", err)
		return nil, fmt.Errorf("failed to verify listing ownership")
	}
	if !ownerResp.IsOwner {
		return nil, fmt.Errorf("seller does not own listing %v", req.ListingId)
	}

	openResp, err := s.listingClient.CheckListingOpen(ctx, &listingproto.CheckListingOpenRequest{
		ListingId: req.ListingId,
	})
	if err != nil {
		log.Printf("Error checking listing status: %v", err)
		return nil, fmt.Errorf("failed to verify listing status")
	}
	if !openResp.IsOpen {
		return nil, fmt.Errorf("listing %v is not available for auction", req.ListingId)
	}

	var auctionExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM auctions WHERE listing_id = $1 AND (status = 'ACTIVE' OR status = 'COMPLETED'))", req.ListingId).Scan(&auctionExists)
	if err != nil {
		log.Printf("Error checking for existing auction: %v", err)
		return nil, fmt.Errorf("failed to check for existing auction")
	}
	if auctionExists {
		return nil, fmt.Errorf("an active or completed auction already exists for listing %v", req.ListingId)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		log.Printf("Error parsing end_time: %v", err)
		return nil, fmt.Errorf("invalid end_time format, expected RFC3339")
	}

	query := `INSERT INTO auctions (listing_id, seller_id, starting_price, reserve_price, status, end_time, created_at, reserve_met, total_bids)
	          VALUES ($1, $2, $3, $4, 'ACTIVE', $5, NOW(), false, 0)
	          RETURNING id, created_at`

	var newId string
	var createdAt time.Time

	err = db.QueryRowContext(ctx, query,
		req.ListingId,
		req.SellerId,
		req.StartingPrice,
		req.ReservePrice,
		endTime,
	).Scan(&newId, &createdAt)

	if err != nil {
		log.Printf("Error in CreateAuction (Insert): %v", err)
		return nil, err
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
	query := `SELECT id, listing_id, seller_id, starting_price, current_price, status, end_time, winner_user_id, created_at, reserve_met, total_bids 
	          FROM auctions WHERE id = $1`

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

	err := db.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &listingId, &sellerId, &startingPrice, &currentPrice,
		&status, &endTime, &winnerUserId, &createdAt, &reserveMet, &totalBids,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not supported/found", req.AuctionId)
			return nil, fmt.Errorf("auction not found")
		}
		log.Printf("Error in GetAuctionDetails (Query): %v", err)
		return nil, err
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
		Status:        status,
		EndTime:       endTime.Format(time.RFC3339),
		WinnerUserId:  winnerUserIdPtr,
		CreatedAt:     createdAt.Format(time.RFC3339),
		ReserveMet:    reserveMet,
		TotalBids:     totalBids,
	}
	return &proto.GetAuctionResponse{Auction: protoAuction}, nil
}

func (s *server) DeleteAuction(ctx context.Context, req *proto.DeleteAuctionRequest) (*proto.DeleteAuctionResponse, error) {
	// 1. Verify if the auction exists
	query := `SELECT id, seller_id FROM auctions WHERE id = $1`

	var (
		id, sellerId string
	)

	err := db.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &sellerId,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not supported/found", req.AuctionId)
			return nil, fmt.Errorf("auction not found")
		}
		log.Printf("Error in DeleteAuction (Query): %v", err)
		return nil, err
	}

	// 2. Verify if the user deleting is the seller
	if sellerId != req.UserId {
		log.Printf("User %s is not the seller of auction %s", req.UserId, req.AuctionId)
		return nil, fmt.Errorf("user not authorized to delete auction")
	}

	// 3. Cancel the auction (soft delete via status)
	_, err = db.ExecContext(ctx, `UPDATE auctions SET status = 'CANCELLED' WHERE id = $1`, req.AuctionId)
	if err != nil {
		log.Printf("Error in DeleteAuction (Update): %v", err)
		return nil, err
	}

	return &proto.DeleteAuctionResponse{Success: true}, nil
}

func (s *server) PlaceBid(ctx context.Context, req *proto.PlaceBidRequest) (*proto.PlaceBidResponse, error) {
	// 1. Verify if the auction exists and is ACTIVE
	query := `SELECT id, seller_id, starting_price, current_price, end_time FROM auctions WHERE id = $1 AND status = 'ACTIVE'`

	var (
		id, sellerId  string
		startingPrice float64
		currentPrice  sql.NullFloat64
		endTime       time.Time
	)

	err := db.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &sellerId, &startingPrice, &currentPrice, &endTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not found or not active", req.AuctionId)
			return nil, fmt.Errorf("auction not found or not active")
		}
		log.Printf("Error in PlaceBid (Query): %v", err)
		return nil, err
	}

	// 2. Bid must be strictly greater than current_price (or starting_price if no bids yet)
	minimumBid := startingPrice
	if currentPrice.Valid {
		minimumBid = currentPrice.Float64
	}
	if req.BidAmount <= minimumBid {
		log.Printf("Bid %f is not greater than minimum %f", req.BidAmount, minimumBid)
		return nil, fmt.Errorf("bid amount must be greater than current price (%.2f)", minimumBid)
	}

	// 3. Insert the bid and get back the generated id
	var (
		newBidID     string
		bidTimestamp time.Time
	)
	err = db.QueryRowContext(ctx,
		`INSERT INTO bids (id, auction_id, bidder_id, bid_amount, timestamp)
		 VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		 RETURNING id, timestamp`,
		req.AuctionId, req.BidderId, req.BidAmount,
	).Scan(&newBidID, &bidTimestamp)
	if err != nil {
		log.Printf("Error in PlaceBid (Insert): %v", err)
		return nil, err
	}

	// 4. Update current_price and total_bids in the auctions table
	_, err = db.ExecContext(ctx,
		`UPDATE auctions SET current_price = $1, total_bids = total_bids + 1 WHERE id = $2`,
		req.BidAmount, req.AuctionId,
	)
	if err != nil {
		log.Printf("Error in PlaceBid (Update): %v", err)
		return nil, err
	}

	// 5. Notify all WebSocket clients watching this auction in real-time
	type BidEvent struct {
		AuctionID string  `json:"auction_id"`
		BidderID  string  `json:"bidder_id"`
		Amount    float64 `json:"amount"`
		Timestamp string  `json:"timestamp"`
	}
	event := BidEvent{
		AuctionID: req.AuctionId,
		BidderID:  req.BidderId,
		Amount:    req.BidAmount,
		Timestamp: bidTimestamp.UTC().Format(time.RFC3339),
	}
	if payload, err := json.Marshal(event); err == nil {
		if pubErr := s.hub.Publish(req.AuctionId, payload); pubErr != nil {
			log.Printf("ws publish error for auction %s: %v", req.AuctionId, pubErr)
		}
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
	// 1. Get total count first
	countQuery := `SELECT COUNT(*) FROM bids WHERE auction_id = $1`
	var total int32
	err := db.QueryRowContext(ctx, countQuery, req.AuctionId).Scan(&total)
	if err != nil {
		log.Printf("Error in GetAuctionBids (Count): %v", err)
		return nil, err
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 10
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// 2. Get all bids for a specific auction, ordered by amount descending
	query := `SELECT id, auction_id, bidder_id, bid_amount, timestamp FROM bids WHERE auction_id = $1 ORDER BY bid_amount DESC LIMIT $2 OFFSET $3`

	rows, err := db.QueryContext(ctx, query, req.AuctionId, limit, offset)
	if err != nil {
		log.Printf("Error in GetAuctionBids (Query): %v", err)
		return nil, err
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
		return nil, err
	}

	return &proto.GetAuctionBidsResponse{
		Bids:     protoBids,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(limit),
	}, nil
}

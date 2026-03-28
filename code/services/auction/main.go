package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	proto "auction/proto"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var db *sql.DB

func initDB() {
	dsn := os.Getenv("AUCTION_DATABASE_URL")
	if dsn == "" {
		log.Fatalf("AUCTION_DATABASE_URL is not set")
	}

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				log.Println("Conectado na base de dados de auctions!")
				return
			}
		}
		log.Printf("A aguardar pela base de dados de auctions... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("failed to connect database: %v", err)
}

type server struct {
	proto.UnimplementedAuctionServiceServer
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

	// TODO: Do calls to listing service to filter by brand, model and location

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
			id, listingId, sellerId, status string
			startingPrice                   float64
			currentPrice                    sql.NullFloat64
			endTime                         time.Time
			winnerUserId                    sql.NullString
			createdAt                       time.Time
			reserveMet                      bool
			totalBids                       int32
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
	// 1. Make a gRPC call to the Listing Service to get listing details
	// 2. Verify if the listing belongs to the seller (user requesting the creation)
	// 3. Verify if the listing is available
	
	// 4. Create the auction in auction-db
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
		id, listingId, sellerId, status string
		startingPrice                   float64
		currentPrice                    sql.NullFloat64
		endTime                         time.Time
		winnerUserId                    sql.NullString
		createdAt                       time.Time
		reserveMet                      bool
		totalBids                       int32
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

	// 3. Delete from auction-db (ou passar status para 'CANCELLED')
	query = `UPDATE auctions SET status = 'CANCELLED' WHERE id = $1`
	err = db.QueryRowContext(ctx, query, req.AuctionId).Scan(&id)
	if err != nil {
		log.Printf("Error in DeleteAuction (Update): %v", err)
		return nil, err
	}

	return &proto.DeleteAuctionResponse{Success: true}, nil
}

func (s *server) PlaceBid(ctx context.Context, req *proto.PlaceBidRequest) (*proto.PlaceBidResponse, error) {
	// 1. Verify if the auction exists and is ACTIVE (end_time > now)
	query := `SELECT id, seller_id, current_price, end_time FROM auctions WHERE id = $1`
	
	var (
		id, sellerId string
		currentPrice float64
		endTime time.Time
	)

	err := db.QueryRowContext(ctx, query, req.AuctionId).Scan(
		&id, &sellerId, &currentPrice, &endTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Auction %s not supported/found", req.AuctionId)
			return nil, fmt.Errorf("auction not found")
		}
		log.Printf("Error in PlaceBid (Query): %v", err)
		return nil, err
	}

	// 2. Verify if the bid amount is strictly greater than current_price
	if req.BidAmount <= currentPrice {
		log.Printf("Bid %f is not greater than current price %f", req.BidAmount, currentPrice)
		return nil, fmt.Errorf("bid amount must be greater than current price")
	}

	// 3. Insert the bid into the bids table
	query = `INSERT INTO bids (auction_id, bidder_id, bid_amount, timestamp) VALUES ($1, $2, $3, $4)`
	err = db.QueryRowContext(ctx, query, req.AuctionId, req.BidderId, req.BidAmount, time.Now()).Scan(&id)
	if err != nil {
		log.Printf("Error in PlaceBid (Insert): %v", err)
		return nil, err
	}

	// 4. Update the current_price in the auctions table
	query = `UPDATE auctions SET current_price = $1 WHERE id = $2`
	err = db.QueryRowContext(ctx, query, req.BidAmount, req.AuctionId).Scan(&id)
	if err != nil {
		log.Printf("Error in PlaceBid (Update): %v", err)
		return nil, err
	}

	return &proto.PlaceBidResponse{}, nil
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
			bidAmount float64
			timestamp time.Time
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

func main() {
	initDB()

	lis, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAuctionServiceServer(grpcServer, &server{})

	log.Println("Auction gRPC server is running on " + lis.Addr().String() + "...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

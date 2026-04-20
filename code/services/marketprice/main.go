package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	marketpricepb "services/marketprice/proto"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var db *sql.DB

type server struct {
	marketpricepb.UnimplementedMarketPriceServiceServer
}

func (s *server) GetAverageMarketPrice(ctx context.Context, req *marketpricepb.AveragePriceRequest) (*marketpricepb.AveragePriceResponse, error) {
	query := `SELECT 
		COALESCE(AVG(ad.ask_price), 0), 
		COALESCE(MIN(ad.ask_price), 0), 
		COALESCE(MAX(ad.ask_price), 0), 
		COUNT(ad.ask_price) 
		FROM automotive_data ad
		JOIN brand b ON ad.brand_id = b.id
		JOIN model m ON ad.model_id = m.id
		WHERE 1=1`

	var args []interface{}
	argId := 1

	if req.Brand != "" {
		query += fmt.Sprintf(` AND b.name = $%d`, argId)
		args = append(args, strings.ToUpper(req.Brand))
		argId++
	}
	if req.Model != "" {
		query += fmt.Sprintf(` AND m.name = $%d`, argId)
		args = append(args, req.Model)
		argId++
	}
	if req.YearFrom != 0 {
		query += fmt.Sprintf(` AND ad.model_year >= $%d`, argId)
		args = append(args, req.YearFrom)
		argId++
	}
	if req.YearTo != 0 {
		query += fmt.Sprintf(` AND ad.model_year <= $%d`, argId)
		args = append(args, req.YearTo)
		argId++
	}

	var avgPrice, minPrice, maxPrice float64
	var count int32

	err := db.QueryRow(query, args...).Scan(&avgPrice, &minPrice, &maxPrice, &count)
	if err != nil {
		log.Printf("Error in query: %v", err)
		return nil, err
	}

	return &marketpricepb.AveragePriceResponse{
		Brand:        req.Brand,
		Model:        req.Model,
		AveragePrice: avgPrice,
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		ListingCount: count,
	}, nil
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL is not set")
	}

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				return
			}
		}
		log.Printf("Waiting for database... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("failed to connect database: %v", err)
}

func main() {
	initDB()

	lis, err := net.Listen("tcp", ":50055")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	marketpricepb.RegisterMarketPriceServiceServer(grpcServer, &server{})

	log.Println("Market Price Analysis gRPC server is running on " + lis.Addr().String() + "...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

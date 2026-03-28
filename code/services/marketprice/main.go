package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	proto "marketprice/proto"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var db *sql.DB

type server struct {
	proto.UnimplementedMarketPriceServiceServer
}

func (s *server) GetAverageMarketPrice(ctx context.Context, req *proto.AveragePriceRequest) (*proto.AveragePriceResponse, error) {
	query := `SELECT 
		COALESCE(AVG("askPrice"), 0), 
		COALESCE(MIN("askPrice"), 0), 
		COALESCE(MAX("askPrice"), 0), 
		COUNT("askPrice") 
		FROM listings WHERE 1=1`

	var args []interface{}
	argId := 1

	if req.Brand != "" {
		query += fmt.Sprintf(` AND "brandName" = $%d`, argId)
		args = append(args, req.Brand)
		argId++
	}
	if req.Model != "" {
		query += fmt.Sprintf(` AND "modelName" = $%d`, argId)
		args = append(args, req.Model)
		argId++
	}
	if req.YearFrom != 0 {
		query += fmt.Sprintf(` AND "vf_ModelYear" >= $%d`, argId)
		args = append(args, req.YearFrom)
		argId++
	}
	if req.YearTo != 0 {
		query += fmt.Sprintf(` AND "vf_ModelYear" <= $%d`, argId)
		args = append(args, req.YearTo)
		argId++
	}

	var avgPrice, minPrice, maxPrice float64
	var count int32

	err := db.QueryRow(query, args...).Scan(&avgPrice, &minPrice, &maxPrice, &count)
	if err != nil {
		log.Printf("Erro na query: %v", err)
		return nil, err
	}

	return &proto.AveragePriceResponse{
		Brand:        req.Brand,
		Model:        req.Model,
		Location:     req.Location,
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
		log.Printf("A aguardar pela base de dados... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("failed to connect database: %v", err)
}

func main() {
	initDB()

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterMarketPriceServiceServer(grpcServer, &server{})

	log.Println("Market Price Analysis gRPC server is running on " + lis.Addr().String() + "...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

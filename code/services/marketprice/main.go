package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	proto "marketprice/proto"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type server struct {
	proto.UnimplementedMarketPriceServiceServer
}

func (s *server) GetAverageMarketPrice(ctx context.Context, req *proto.AveragePriceRequest) (*proto.AveragePriceResponse, error) {
	query := db.Table("listings")

	if req.Brand != "" {
		query = query.Where(`"brandName" = ?`, req.Brand)
	}
	if req.Model != "" {
		query = query.Where(`"modelName" = ?`, req.Model)
	}
	if req.YearFrom != 0 {
		query = query.Where(`"vf_ModelYear" >= ?`, req.YearFrom)
	}
	if req.YearTo != 0 {
		query = query.Where(`"vf_ModelYear" <= ?`, req.YearTo)
	}

	var avgPrice, minPrice, maxPrice float64
	var count int32

	row := query.Select(
		"COALESCE(AVG(\"askPrice\"), 0)", 
		"COALESCE(MIN(\"askPrice\"), 0)", 
		"COALESCE(MAX(\"askPrice\"), 0)", 
		"COUNT(\"askPrice\")",
	).Row()
	
	err := row.Scan(&avgPrice, &minPrice, &maxPrice, &count)
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
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, _ := db.DB()
			if pingErr := sqlDB.Ping(); pingErr == nil {
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

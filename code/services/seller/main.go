package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"

	. "seller/models"
	proto "seller/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type server struct {
	proto.UnimplementedSellerServiceServer
}

func (s *server) GetSellerProfile(ctx context.Context, req *proto.GetSellerProfileRequest) (*proto.SellerProfileResponse, error) {
	if req.SellerId == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}

	var seller Seller
	if err := db.Where("id = ?", req.SellerId).First(&seller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "seller not found")
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	return &proto.SellerProfileResponse{
		SellerId:    seller.ID,
		Name:        seller.Name,
		SellerType:  seller.SellerType,
		ContactInfo: seller.ContactInfo,
		Rating:      seller.Rating,
	}, nil
}

func (s *server) VerifySellerProfile(ctx context.Context, req *proto.VerifySellerRequest) (*proto.VerifySellerResponse, error) {
	if req.SellerId == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}

	var seller Seller
	if err := db.Where("id = ?", req.SellerId).First(&seller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.VerifySellerResponse{IsSeller: false}, nil
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	return &proto.VerifySellerResponse{IsSeller: seller.IsSeller}, nil
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL is not set")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	db.AutoMigrate(&Seller{})
}

func main() {
	initDB()

	lis, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterSellerServiceServer(grpcServer, &server{})

	log.Println("Seller gRPC server is running on " + lis.Addr().String() + "...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

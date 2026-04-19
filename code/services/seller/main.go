package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	. "services/seller/models"
	proto "services/seller/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var listingDb *gorm.DB

type server struct {
	proto.UnimplementedSellerServiceServer
}

func (s *server) CreateSeller(ctx context.Context, req *proto.CreateSellerRequest) (*proto.SellerProfileResponse, error) {
	if req.SellerId <= 0 || req.Name == "" || req.SellerType == "" || req.ContactInfo == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id, name, seller_type, and contact_info are required")
	}

	if req.SellerType != "professional_dealer" && req.SellerType != "private_seller" {
		return nil, status.Error(codes.InvalidArgument, "seller_type must be either 'professional_dealer' or 'private_seller'")
	}

	var existing Seller
	if err := db.Where("id = ?", req.SellerId).First(&existing).Error; err == nil {
		return nil, status.Error(codes.AlreadyExists, "seller already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, status.Error(codes.Internal, "database error")
	}

	seller := Seller{
		ID:          req.SellerId,
		Name:        req.Name,
		SellerType:  req.SellerType,
		ContactInfo: req.ContactInfo,
		Rating:      0,
	}

	if err := db.Create(&seller).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to create seller")
	}

	return &proto.SellerProfileResponse{
		SellerId:    seller.ID,
		Name:        seller.Name,
		SellerType:  seller.SellerType,
		ContactInfo: seller.ContactInfo,
		Rating:      seller.Rating,
	}, nil
}

func (s *server) CreateListing(ctx context.Context, req *proto.CreateListingRequest) (*proto.CreateListingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "listing details are required")
	}
	if req.Vin == "" {
		return nil, status.Error(codes.InvalidArgument, "vin is required")
	}
	if req.Make == "" || req.Model == "" {
		return nil, status.Error(codes.InvalidArgument, "make and model are required")
	}
	if req.DealerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "dealer_id must be a positive integer")
	}
	if listingDb == nil {
		return nil, status.Error(codes.Internal, "listing database is not configured")
	}

	listingID, listedAt, err := createListing(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create listing: %v", err)
	}

	listedAtText := ""
	if !listedAt.IsZero() {
		listedAtText = listedAt.UTC().Format(time.RFC3339)
	}

	return &proto.CreateListingResponse{
		Id:       listingID,
		ListedAt: listedAtText,
	}, nil
}

func (s *server) GetSellerProfile(ctx context.Context, req *proto.GetSellerProfileRequest) (*proto.SellerProfileResponse, error) {
	if req.SellerId <= 0 {
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
	if req.SellerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}

	var seller Seller
	if err := db.Where("id = ?", req.SellerId).First(&seller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.VerifySellerResponse{IsSeller: false}, nil
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	return &proto.VerifySellerResponse{IsSeller: true}, nil
}

func (s *server) GetSellersPreview(ctx context.Context, req *proto.SellersPreviewRequest) (*proto.SellersPreviewResponse, error) {
	if len(req.SellerIds) == 0 {
		return &proto.SellersPreviewResponse{Sellers: []*proto.SellerPreview{}}, nil
	}

	var sellers []Seller
	if err := db.Where("id IN ?", req.SellerIds).Find(&sellers).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get sellers preview")
	}

	previews := make([]*proto.SellerPreview, 0, len(sellers))
	for _, seller := range sellers {
		previews = append(previews, &proto.SellerPreview{
			Id:   seller.ID,
			Name: seller.Name,
		})
	}

	return &proto.SellersPreviewResponse{
		Sellers: previews,
	}, nil
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
	initListingDB()

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

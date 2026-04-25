package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	. "services/seller/models"
	proto "services/seller/proto"
	"services/seller/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	proto.SellerServiceServer
	service *service.Service
}

func (s *server) CreateSeller(ctx context.Context, req *proto.CreateSellerRequest) (*proto.SellerProfileResponse, error) {
	if req.SellerId <= 0 || req.Name == "" || req.SellerType == "" || req.ContactInfo == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id, name, seller_type, and contact_info are required")
	}

	if req.SellerType != "professional_dealer" && req.SellerType != "private_seller" {
		return nil, status.Error(codes.InvalidArgument, "seller_type must be either 'professional_dealer' or 'private_seller'")
	}
	return s.service.CreateSeller(ctx, req)
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

	return s.service.CreateListing(ctx, req)
}

func (s *server) GetSellerProfile(ctx context.Context, req *proto.GetSellerProfileRequest) (*proto.SellerProfileResponse, error) {
	if req.SellerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}
	return s.service.GetSellerProfile(ctx, req)
}

func (s *server) VerifySellerProfile(ctx context.Context, req *proto.VerifySellerRequest) (*proto.VerifySellerResponse, error) {
	if req.SellerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}
	return s.service.VerifySellerProfile(ctx, req)
}

func (s *server) GetSellersPreview(ctx context.Context, req *proto.SellersPreviewRequest) (*proto.SellersPreviewResponse, error) {
	if len(req.SellerIds) == 0 {
		return &proto.SellersPreviewResponse{Sellers: []*proto.SellerPreview{}}, nil
	}
	return s.service.GetSellersPreview(ctx, req)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Error("DATABASE_URL is not set")
		return
	}
	listingDsn := os.Getenv("LISTING_DATABASE_URL")
	if listingDsn == "" {
		logger.Error("LISTING_DATABASE_URL is not set")
		return
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		return
	}
	if err := db.AutoMigrate(&Seller{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	listingDB, err := gorm.Open(postgres.Open(listingDsn), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect listing database", "error", err)
		return
	}

	port := os.Getenv("SELLER_GRPC_PORT")
	if port == "" {
		logger.Error("SELLER_GRPC_PORT is not set")
		return
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("error on listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer()
	sellerSvc := service.NewService(db, listingDB)
	proto.RegisterSellerServiceServer(grpcServer, &server{service: sellerSvc})

	logger.Info("Seller gRPC server is running", "addr", lis.Addr().String())

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}

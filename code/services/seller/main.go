package main

import (
	"context"
	"log/slog"
	"os"

	sellerpb "services/seller/proto"
	"services/seller/repository"
	"services/seller/service"
	"services/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	sellerpb.SellerServiceServer
	service *service.Service
}

func (s *server) CreateSeller(ctx context.Context, req *sellerpb.CreateSellerRequest) (*sellerpb.SellerProfileResponse, error) {
	if req.SellerId <= 0 || req.Name == "" || req.SellerType == "" || req.ContactInfo == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id, name, seller_type, and contact_info are required")
	}
	if req.SellerType != "professional_dealer" && req.SellerType != "private_seller" {
		return nil, status.Error(codes.InvalidArgument, "seller_type must be either 'professional_dealer' or 'private_seller'")
	}
	return s.service.CreateSeller(ctx, req)
}

func (s *server) CreateListing(ctx context.Context, req *sellerpb.CreateListingRequest) (*sellerpb.CreateListingResponse, error) {
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

func (s *server) GetSellerProfile(ctx context.Context, req *sellerpb.GetSellerProfileRequest) (*sellerpb.SellerProfileResponse, error) {
	if req.SellerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}
	return s.service.GetSellerProfile(ctx, req)
}

func (s *server) VerifySellerProfile(ctx context.Context, req *sellerpb.VerifySellerRequest) (*sellerpb.VerifySellerResponse, error) {
	if req.SellerId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}
	return s.service.VerifySellerProfile(ctx, req)
}

func (s *server) GetSellersPreview(ctx context.Context, req *sellerpb.SellersPreviewRequest) (*sellerpb.SellersPreviewResponse, error) {
	if len(req.SellerIds) == 0 {
		return &sellerpb.SellersPreviewResponse{Sellers: []*sellerpb.SellerPreview{}}, nil
	}

	ids := make([]int64, 0, len(req.SellerIds))

	previews, err := s.service.GetSellersPreview(ctx, ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get sellers preview")
	}

	return &sellerpb.SellersPreviewResponse{
		Sellers: previews,
	}, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	sellerDsn := utils.MustGetEnv("SELLER_DATABASE_URL")
	listingDsn := utils.MustGetEnv("LISTING_DATABASE_URL")

	sellerDB := utils.TryConnectGorm(sellerDsn, 3, 10)
	listingDB := utils.TryConnectGorm(listingDsn, 3, 10)

	sellerGrpcPort := utils.MustGetEnv("SELLER_GRPC_PORT")

	lis := utils.TryListen(sellerGrpcPort)

	grpcServer := grpc.NewServer()
	repo := repository.NewRepository(sellerDB, listingDB)
	sellerSvc := service.NewService(repo)
	sellerpb.RegisterSellerServiceServer(grpcServer, &server{service: sellerSvc})

	utils.HealthCheck("seller.SellerService", grpcServer)

	logger.Info("Seller gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}

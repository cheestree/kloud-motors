package service

import (
	"context"
	"errors"
	"time"

	. "services/seller/models"
	proto "services/seller/proto"
	"services/seller/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Service struct {
	DB        *gorm.DB
	ListingDB *gorm.DB
}

func NewService(db *gorm.DB, listingDB *gorm.DB) *Service {
	return &Service{DB: db, ListingDB: listingDB}
}

func (s *Service) CreateSeller(ctx context.Context, req *proto.CreateSellerRequest) (*proto.SellerProfileResponse, error) {
	var existing Seller
	if err := s.DB.Where("id = ?", req.SellerId).First(&existing).Error; err == nil {
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

	if err := s.DB.Create(&seller).Error; err != nil {
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

func (s *Service) CreateListing(ctx context.Context, req *proto.CreateListingRequest) (*proto.CreateListingResponse, error) {
	listingID, listedAt, err := repository.CreateListing(ctx, s.ListingDB, req)
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

func (s *Service) GetSellerProfile(ctx context.Context, req *proto.GetSellerProfileRequest) (*proto.SellerProfileResponse, error) {
	var seller Seller
	if err := s.DB.Where("id = ?", req.SellerId).First(&seller).Error; err != nil {
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

func (s *Service) VerifySellerProfile(ctx context.Context, req *proto.VerifySellerRequest) (*proto.VerifySellerResponse, error) {
	var seller Seller
	if err := s.DB.Where("id = ?", req.SellerId).First(&seller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &proto.VerifySellerResponse{IsSeller: false}, nil
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	return &proto.VerifySellerResponse{IsSeller: true}, nil
}

func (s *Service) GetSellersPreview(ctx context.Context, req *proto.SellersPreviewRequest) (*proto.SellersPreviewResponse, error) {
	var sellers []Seller
	if err := s.DB.Where("id IN ?", req.SellerIds).Find(&sellers).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get sellers preview")
	}

	previews := make([]*proto.SellerPreview, 0, len(sellers))
	for _, seller := range sellers {
		previews = append(previews, &proto.SellerPreview{
			Id:   seller.ID,
			Name: seller.Name,
		})
	}

	return &proto.SellersPreviewResponse{Sellers: previews}, nil
}

package service

import (
	"context"

	. "services/user/models"
	proto "services/user/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) CreateUserProfile(ctx context.Context, req *proto.CreateUserProfileRequest) (*proto.CreateUserProfileResponse, error) {
	newUser := User{
		ID:    req.UserId,
		Name:  req.Name,
		Email: req.Email,
	}

	if err := s.DB.Create(&newUser).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to create user profile")
	}

	return &proto.CreateUserProfileResponse{Success: true}, nil
}

func (s *Service) GetFavorites(ctx context.Context, req *proto.GetFavoritesRequest) (*proto.FavoritesResponse, error) {
	var favorites []Favorite
	if err := s.DB.Where("user_id = ?", req.UserId).Find(&favorites).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get favorites")
	}

	listings := make([]int64, len(favorites))
	for i, f := range favorites {
		listings[i] = f.ListingID
	}

	return &proto.FavoritesResponse{Favorites: listings}, nil
}

func (s *Service) AddFavorite(ctx context.Context, req *proto.AddFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	fav := Favorite{
		UserID:    req.UserId,
		ListingID: req.ListingId,
	}

	if err := s.DB.Create(&fav).Error; err != nil {
		return &proto.FavoriteMutationResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.FavoriteMutationResponse{
		Success: true,
		Message: "favorite added",
	}, nil
}

func (s *Service) RemoveFavorite(ctx context.Context, req *proto.RemoveFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	res := s.DB.Where("user_id = ? AND listing_id = ?", req.UserId, req.ListingId).Delete(&Favorite{})
	if res.Error != nil {
		return &proto.FavoriteMutationResponse{
			Success: false,
			Message: res.Error.Error(),
		}, nil
	}

	if res.RowsAffected == 0 {
		return &proto.FavoriteMutationResponse{
			Success: false,
			Message: "favorite not found",
		}, nil
	}

	return &proto.FavoriteMutationResponse{
		Success: true,
		Message: "favorite removed",
	}, nil
}

func (s *Service) GetUsersPreview(ctx context.Context, req *proto.UsersPreviewRequest) (*proto.UsersPreviewResponse, error) {
	var users []User
	if err := s.DB.Where("id IN ?", req.UserIds).Find(&users).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get users preview")
	}

	previews := make([]*proto.UserPreview, 0, len(users))
	for _, user := range users {
		previews = append(previews, &proto.UserPreview{
			Id:   user.ID,
			Name: user.Name,
		})
	}

	return &proto.UsersPreviewResponse{Users: previews}, nil
}

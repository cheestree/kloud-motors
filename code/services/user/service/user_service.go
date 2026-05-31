package service

import (
	"context"
	"strings"

	. "services/user/models"
	proto "services/user/proto"
	"services/user/repository"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	repo *repository.Repository
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetOrCreateByFirebaseUID(ctx context.Context, req *proto.GetOrCreateByFirebaseUIDRequest) (*proto.GetOrCreateByFirebaseUIDResponse, error) {
	user, err := s.repo.GetOrCreateByFirebaseUID(ctx, req.FirebaseUid, req.Email, req.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get or create user by firebase uid")
	}
	return &proto.GetOrCreateByFirebaseUIDResponse{UserId: user.ID}, nil
}

func (s *UserService) CreateUserProfile(ctx context.Context, req *proto.CreateUserProfileRequest) (*proto.CreateUserProfileResponse, error) {
	user := &User{
		ID:    req.UserId,
		Name:  req.Name,
		Email: req.Email,
	}

	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create user profile")
	}

	return &proto.CreateUserProfileResponse{Success: true}, nil
}

func (s *UserService) GetFavorites(ctx context.Context, req *proto.GetFavoritesRequest) (*proto.FavoritesResponse, error) {
	favs, err := s.repo.GetFavorites(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get favorites")
	}

	listings := make([]int64, len(favs))
	for i, f := range favs {
		listings[i] = f.ListingID
	}

	return &proto.FavoritesResponse{
		Favorites: listings,
	}, nil
}

func (s *UserService) AddFavorite(ctx context.Context, req *proto.AddFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	fav := &Favorite{
		UserID:    req.UserId,
		ListingID: req.ListingId,
	}

	err := s.repo.AddFavorite(ctx, fav)
	if err != nil {
		if isDuplicateFavoriteError(err) {
			return nil, status.Error(codes.AlreadyExists, "favorite already exists")
		}
		return nil, status.Error(codes.Internal, "failed to add favorite")
	}

	return &proto.FavoriteMutationResponse{
		Success: true,
		Message: "favorite added",
	}, nil
}

func isDuplicateFavoriteError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "idx_user_listing")
}

func (s *UserService) RemoveFavorite(ctx context.Context, req *proto.RemoveFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	rows, err := s.repo.RemoveFavorite(ctx, req.UserId, req.ListingId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to remove favorite")
	}

	if rows == 0 {
		return nil, status.Error(codes.NotFound, "favorite not found")
	}

	return &proto.FavoriteMutationResponse{
		Success: true,
		Message: "favorite removed",
	}, nil
}

func (s *UserService) GetUsersPreview(ctx context.Context, req *proto.UsersPreviewRequest) (*proto.UsersPreviewResponse, error) {
	users, err := s.repo.GetUsersByIDs(ctx, req.UserIds)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get users preview")
	}

	previews := make([]*proto.UserPreview, 0, len(users))
	for _, u := range users {
		previews = append(previews, &proto.UserPreview{
			Id:   u.ID,
			Name: u.Name,
		})
	}

	return &proto.UsersPreviewResponse{
		Users: previews,
	}, nil
}

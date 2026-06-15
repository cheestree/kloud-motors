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
	auth *firebaseAuthClient
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{
		repo: repo,
		auth: newFirebaseAuthClientFromEnv(),
	}
}

func (s *UserService) Login(ctx context.Context, req *proto.AuthRequest) (*proto.AuthResponse, error) {
	authResp, err := s.auth.login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	return s.withUserID(ctx, authResp)
}

func (s *UserService) Register(ctx context.Context, req *proto.AuthRequest) (*proto.AuthResponse, error) {
	authResp, err := s.auth.register(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}
	return s.withUserID(ctx, authResp)
}

func (s *UserService) RefreshToken(ctx context.Context, req *proto.RefreshTokenRequest) (*proto.AuthResponse, error) {
	authResp, err := s.auth.refreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	return s.withUserID(ctx, authResp)
}

func (s *UserService) withUserID(ctx context.Context, authResp *proto.AuthResponse) (*proto.AuthResponse, error) {
	user, err := s.repo.GetOrCreateByFirebaseUID(ctx, authResp.LocalId, authResp.Email, "")
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to resolve authenticated user")
	}
	authResp.UserId = user.ID
	return authResp, nil
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

package main

import (
	"context"
	"log/slog"
	"os"

	"services/user/models"
	userpb "services/user/proto"
	"services/user/service"
	"services/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	userpb.UserServiceServer
	service *service.Service
}

func (s *server) CreateUserProfile(ctx context.Context, req *userpb.CreateUserProfileRequest) (*userpb.CreateUserProfileResponse, error) {
	if req.UserId <= 0 || req.Name == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, name, and email are required")
	}

	return s.service.CreateUserProfile(ctx, req)
}

func (s *server) GetFavorites(ctx context.Context, req *userpb.GetFavoritesRequest) (*userpb.FavoritesResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	return s.service.GetFavorites(ctx, req)
}

func (s *server) AddFavorite(ctx context.Context, req *userpb.AddFavoriteRequest) (*userpb.FavoriteMutationResponse, error) {
	if req.UserId <= 0 || req.ListingId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}

	return s.service.AddFavorite(ctx, req)
}

func (s *server) RemoveFavorite(ctx context.Context, req *userpb.RemoveFavoriteRequest) (*userpb.FavoriteMutationResponse, error) {
	if req.UserId <= 0 || req.ListingId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}
	return s.service.RemoveFavorite(ctx, req)
}

func (s *server) GetUsersPreview(ctx context.Context, req *userpb.UsersPreviewRequest) (*userpb.UsersPreviewResponse, error) {
	if len(req.UserIds) == 0 {
		return &userpb.UsersPreviewResponse{Users: []*userpb.UserPreview{}}, nil
	}
	return s.service.GetUsersPreview(ctx, req)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	userDsn := utils.MustGetEnv("USER_DATABASE_URL")

	userDB := utils.TryConnectGorm(userDsn, 3, 10)
	if err := userDB.AutoMigrate(&models.User{}, &models.Favorite{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	userGrpcPort := utils.MustGetEnv("USER_GRPC_PORT")

	lis := utils.TryListen(userGrpcPort)

	grpcServer := grpc.NewServer()
	userSvc := service.NewService(userDB)
	userpb.RegisterUserServiceServer(grpcServer, &server{service: userSvc})

	utils.HealthCheck("user.UserService", grpcServer)

	logger.Info("User gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}

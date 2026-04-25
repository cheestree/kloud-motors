package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	. "services/user/models"
	proto "services/user/proto"
	"services/user/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	proto.UserServiceServer
	service *service.Service
}

func (s *server) CreateUserProfile(ctx context.Context, req *proto.CreateUserProfileRequest) (*proto.CreateUserProfileResponse, error) {
	if req.UserId <= 0 || req.Name == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, name, and email are required")
	}

	return s.service.CreateUserProfile(ctx, req)
}

func (s *server) GetFavorites(ctx context.Context, req *proto.GetFavoritesRequest) (*proto.FavoritesResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	return s.service.GetFavorites(ctx, req)
}

func (s *server) AddFavorite(ctx context.Context, req *proto.AddFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	if req.UserId <= 0 || req.ListingId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}

	return s.service.AddFavorite(ctx, req)
}

func (s *server) RemoveFavorite(ctx context.Context, req *proto.RemoveFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	if req.UserId <= 0 || req.ListingId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}
	return s.service.RemoveFavorite(ctx, req)
}

func (s *server) GetUsersPreview(ctx context.Context, req *proto.UsersPreviewRequest) (*proto.UsersPreviewResponse, error) {
	if len(req.UserIds) == 0 {
		return &proto.UsersPreviewResponse{Users: []*proto.UserPreview{}}, nil
	}
	return s.service.GetUsersPreview(ctx, req)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Error("DATABASE_URL is not set")
		return
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		return
	}
	if err := db.AutoMigrate(&User{}, &Favorite{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	port := os.Getenv("USER_GRPC_PORT")
	if port == "" {
		logger.Error("USER_GRPC_PORT is not set")
		return
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Error("error on listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer()
	userSvc := service.NewService(db)
	proto.RegisterUserServiceServer(grpcServer, &server{service: userSvc})

	logger.Info("User gRPC server is running", "addr", lis.Addr().String())

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}

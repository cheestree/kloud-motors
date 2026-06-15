package main

import (
	"context"
	"log/slog"
	"os"

	"services/observability"
	"services/user/models"
	userpb "services/user/proto"
	"services/user/repository"
	"services/user/service"
	"services/utils"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	userpb.UserServiceServer
	service *service.UserService
}

func (s *server) Login(ctx context.Context, req *userpb.AuthRequest) (*userpb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	return s.service.Login(ctx, req)
}

func (s *server) Register(ctx context.Context, req *userpb.AuthRequest) (*userpb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	return s.service.Register(ctx, req)
}

func (s *server) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.AuthResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}
	return s.service.RefreshToken(ctx, req)
}

func (s *server) GetOrCreateByFirebaseUID(ctx context.Context, req *userpb.GetOrCreateByFirebaseUIDRequest) (*userpb.GetOrCreateByFirebaseUIDResponse, error) {
	if req.FirebaseUid == "" {
		return nil, status.Error(codes.InvalidArgument, "firebase_uid is required")
	}
	return s.service.GetOrCreateByFirebaseUID(ctx, req)
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
	ctx := context.Background()
	shutdownTracing := observability.InitTracing(ctx, logger, "user")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	userDsn := utils.MustGetEnv("USER_DATABASE_URL")

	userDB := utils.TryConnectGorm(userDsn, 8, 10)
	if err := userDB.AutoMigrate(&models.User{}, &models.Favorite{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	userGrpcPort := utils.MustGetEnv("USER_GRPC_PORT")

	lis := utils.TryListen(userGrpcPort)

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	repo := repository.NewRepository(userDB)
	userSvc := service.NewUserService(repo)
	userpb.RegisterUserServiceServer(grpcServer, &server{service: userSvc})

	utils.HealthCheck("user.UserService", grpcServer)

	logger.Info("User gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}

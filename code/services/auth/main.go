package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"os"

	authpb "services/auth/proto"
	"services/auth/models"
	"services/auth/repository"
	"services/auth/service"
	"services/utils"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	authpb.AuthServiceServer
	service *service.AuthService
}

func getPrivateKey() (interface{}, error) {
	b64Key := os.Getenv("JWT_PRIVATE_KEY_B64")
	if b64Key == "" {
		return nil, errors.New("JWT_PRIVATE_KEY_B64 is not configured")
	}
	keyBytes, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, errors.New("failed to decode base64 private key")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, errors.New("failed to parse RSA private key")
	}
	return key, nil
}

func (s *server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	return s.service.Register(ctx, req)
}

func (s *server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "missing credentials")
	}
	return s.service.Login(ctx, req)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	authDSN := utils.MustGetEnv("AUTH_DATABASE_URL")

	authGrpcPort := utils.MustGetEnv("AUTH_GRPC_PORT")

	authDB := utils.TryConnectGorm(authDSN, 8, 10)
	if err := authDB.AutoMigrate(&models.AuthUser{}); err != nil {
		logger.Error("failed to migrate auth database", "error", err)
		return
	}

	privateKey, err := getPrivateKey()
	if err != nil {
		logger.Error("failed to load private key", "error", err)
		return
	}

	lis := utils.TryListen(authGrpcPort)

	grpcServer := grpc.NewServer()
	repo := repository.NewAuthRepository(authDB)
	authSvc := service.NewAuthService(repo, privateKey)
	authpb.RegisterAuthServiceServer(grpcServer, &server{service: authSvc})

	utils.HealthCheck("auth.AuthService", grpcServer)

	logger.Info("Auth gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}

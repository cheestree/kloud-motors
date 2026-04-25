package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"os"

	. "services/auth/models"
	proto "services/auth/proto"
	"services/auth/repository"
	"services/auth/service"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type server struct {
	proto.AuthServiceServer
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

func (s *server) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	return s.service.Register(ctx, req)
}

func (s *server) Login(ctx context.Context, req *proto.LoginRequest) (*proto.AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}
	return s.service.Login(ctx, req)
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
	if err := db.AutoMigrate(&AuthUser{}); err != nil {
		logger.Error("failed to migrate database", "error", err)
		return
	}

	privateKey, err := getPrivateKey()
	if err != nil {
		logger.Error("failed to load private key", "error", err)
		return
	}
	repo := repository.NewAuthRepository(db)
	authService := service.NewAuthService(repo, privateKey)

	grpcPort := os.Getenv("AUTH_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("AUTH_GRPC_PORT is not set")
		return
	}
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("error on listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAuthServiceServer(grpcServer, &server{service: authService})

	logger.Info("Auth gRPC server is running", "addr", lis.Addr().String())

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}

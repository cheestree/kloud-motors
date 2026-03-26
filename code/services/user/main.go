package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"

	. "user/models"
	proto "user/proto"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)
type server struct {
	proto.UnimplementedUserServiceServer
}

func CheckUserExists(email string) bool {
	var user User
	if err := db.Where("email = ?", email).First(&user).Error; err == nil {
		return true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	return false
}

func (s *server) RegisterUser(ctx context.Context, req *proto.RegisterUserRequest) (*proto.AuthResponse, error) {
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "name, email, and password are required")
	}

	if CheckUserExists(req.Email) {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	newUser := User{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &proto.AuthResponse{
		UserId: newUser.ID,
		Token:  "token-placeholder", // Token generation not in scope, just returning dummy
	}, nil
}

func (s *server) LoginUser(ctx context.Context, req *proto.LoginUserRequest) (*proto.AuthResponse, error) {
	var user User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return &proto.AuthResponse{
		UserId: user.ID,
		Token:  "token-placeholder",
	}, nil
}

func (s *server) CheckUserExists(ctx context.Context, req *proto.CheckUserExistsRequest) (*proto.CheckUserExistsResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	exists := CheckUserExists(req.Email)
	return &proto.CheckUserExistsResponse{Exists: exists}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterUserServiceServer(grpcServer, &server{})

	log.Println("User gRPC server is running on " + lis.Addr().String() + "...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

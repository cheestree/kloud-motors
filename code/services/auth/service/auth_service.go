package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	. "services/auth/models"
	authpb "services/auth/proto"
	"services/auth/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthService struct {
	DB         *repository.AuthRepository
	PrivateKey interface{}
}

func NewAuthService(repo *repository.AuthRepository, privateKey interface{}) *AuthService {
	return &AuthService{DB: repo, PrivateKey: privateKey}
}

type UserClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateJWT(user *AuthUser) (string, error) {
	claims := UserClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.PrivateKey)
}

func (s *AuthService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	user, err := s.DB.GetUserByEmail(req.Email)

	if user != nil || err == nil {
		return nil, status.Error(codes.AlreadyExists, "user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	newUser := AuthUser{
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := s.DB.CreateUser(&newUser); err != nil {
		return nil, status.Error(codes.Internal, "failed to create user auth record")
	}

	token, err := s.GenerateJWT(&newUser)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &authpb.AuthResponse{
		UserId: newUser.ID,
		Token:  token,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	user, err := s.DB.GetUserByEmail(req.Email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	token, err := s.GenerateJWT(user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &authpb.AuthResponse{
		UserId: user.ID,
		Token:  token,
	}, nil
}

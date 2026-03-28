package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	. "user/models"
	proto "user/proto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type server struct {
	proto.UnimplementedUserServiceServer
}

type UserClaims struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	IsSeller   bool   `json:"is_seller"`
	SellerType string `json:"seller_type,omitempty"`
	jwt.RegisteredClaims
}

func generateJWT(user *User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", status.Error(codes.Internal, "JWT_SECRET is not configured")
	}

	claims := UserClaims{
		UserID:     user.ID,
		Email:      user.Email,
		IsSeller:   user.IsSeller,
		SellerType: user.SellerType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
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

	if req.IsSeller {
		if req.SellerType != "professional_dealer" && req.SellerType != "private_seller" {
			return nil, status.Error(codes.InvalidArgument, "seller_type must be either 'professional_dealer' or 'private_seller' for sellers")
		}
	} else if req.SellerType != "" {
		return nil, status.Error(codes.InvalidArgument, "seller_type should not be provided for non-sellers")
	}

	newUser := User{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Email:       req.Email,
		Password:    string(hashedPassword),
		IsSeller:    req.IsSeller,
		SellerType:  req.SellerType,
		ContactInfo: req.ContactInfo,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	token, err := generateJWT(&newUser)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &proto.AuthResponse{
		UserId: newUser.ID,
		Token:  token,
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

	token, err := generateJWT(&user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &proto.AuthResponse{
		UserId: user.ID,
		Token:  token,
	}, nil
}

func (s *server) GetFavorites(ctx context.Context, req *proto.GetFavoritesRequest) (*proto.FavoritesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var favorites []Favorite
	if err := db.Where("user_id = ?", req.UserId).Find(&favorites).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get favorites")
	}

	listings := make([]string, len(favorites))
	for i, f := range favorites {
		listings[i] = f.ListingID
	}

	return &proto.FavoritesResponse{
		Favorites: listings,
	}, nil
}

func (s *server) AddFavorite(ctx context.Context, req *proto.AddFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	if req.UserId == "" || req.ListingId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}

	fav := Favorite{
		UserID:    req.UserId,
		ListingID: req.ListingId,
	}

	if err := db.Create(&fav).Error; err != nil {
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

func (s *server) RemoveFavorite(ctx context.Context, req *proto.RemoveFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	if req.UserId == "" || req.ListingId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and listing_id are required")
	}

	res := db.Where("user_id = ? AND listing_id = ?", req.UserId, req.ListingId).Delete(&Favorite{})
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

func (s *server) CheckUserExists(ctx context.Context, req *proto.CheckUserExistsRequest) (*proto.CheckUserExistsResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	exists := CheckUserExists(req.Email)
	return &proto.CheckUserExistsResponse{Exists: exists}, nil
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatalf("DATABASE_URL is not set")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	db.AutoMigrate(&User{}, &Favorite{})
}

func main() {
	initDB()

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

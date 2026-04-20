package main

import (
	"context"
	"log"
	"net"
	"os"

	. "services/user/models"
	proto "services/user/proto"

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

func (s *server) CreateUserProfile(ctx context.Context, req *proto.CreateUserProfileRequest) (*proto.CreateUserProfileResponse, error) {
	if req.UserId <= 0 || req.Name == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, name, and email are required")
	}

	newUser := User{
		ID:    req.UserId,
		Name:  req.Name,
		Email: req.Email,
	}

	if err := db.Create(&newUser).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to create user profile")
	}

	return &proto.CreateUserProfileResponse{
		Success: true,
	}, nil
}

func (s *server) GetFavorites(ctx context.Context, req *proto.GetFavoritesRequest) (*proto.FavoritesResponse, error) {
	if req.UserId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var favorites []Favorite
	if err := db.Where("user_id = ?", req.UserId).Find(&favorites).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get favorites")
	}

	listings := make([]int64, len(favorites))
	for i, f := range favorites {
		listings[i] = f.ListingID
	}

	return &proto.FavoritesResponse{
		Favorites: listings,
	}, nil
}

func (s *server) AddFavorite(ctx context.Context, req *proto.AddFavoriteRequest) (*proto.FavoriteMutationResponse, error) {
	if req.UserId <= 0 || req.ListingId <= 0 {
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
	if req.UserId <= 0 || req.ListingId <= 0 {
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

func (s *server) GetUsersPreview(ctx context.Context, req *proto.UsersPreviewRequest) (*proto.UsersPreviewResponse, error) {
	if len(req.UserIds) == 0 {
		return &proto.UsersPreviewResponse{Users: []*proto.UserPreview{}}, nil
	}

	var users []User
	if err := db.Where("id IN ?", req.UserIds).Find(&users).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get users preview")
	}

	previews := make([]*proto.UserPreview, 0, len(users))
	for _, user := range users {
		previews = append(previews, &proto.UserPreview{
			Id:   user.ID,
			Name: user.Name,
		})
	}

	return &proto.UsersPreviewResponse{
		Users: previews,
	}, nil
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

	lis, err := net.Listen("tcp", ":50058")
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

package repository

import (
	models "services/auth/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type AuthRepository struct {
	db gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: *db}
}

func (r *AuthRepository) GetUserByEmail(email string) (*models.AuthUser, error) {
	var user models.AuthUser
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &user, nil
}


func (r *AuthRepository) CreateUser(user *models.AuthUser) error {
	if err := r.db.Create(&user).Error; err != nil {
		return status.Error(codes.Internal, "failed to create user auth record")
	}
	return nil
}

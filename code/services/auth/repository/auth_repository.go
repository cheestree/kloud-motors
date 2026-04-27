package repository

import (
	models "services/auth/models"

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

	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) CreateUser(user *models.AuthUser) error {
	return r.db.Create(user).Error
}

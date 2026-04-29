package repository

import (
	"context"
	"services/user/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetFavorites(ctx context.Context, userID int64) ([]models.Favorite, error) {
	var favs []models.Favorite

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&favs).Error

	return favs, err
}

func (r *Repository) AddFavorite(ctx context.Context, fav *models.Favorite) error {
	return r.db.WithContext(ctx).Create(fav).Error
}

func (r *Repository) RemoveFavorite(ctx context.Context, userID, listingID int64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND listing_id = ?", userID, listingID).
		Delete(&models.Favorite{})

	return res.RowsAffected, res.Error
}

func (r *Repository) GetUsersByIDs(ctx context.Context, ids []int64) ([]models.User, error) {
	var users []models.User

	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&users).Error

	return users, err
}

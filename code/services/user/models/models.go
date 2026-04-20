package models

type User struct {
	ID          int64 `gorm:"primaryKey"`
	Name        string
	Email       string `gorm:"uniqueIndex"`
}

type Favorite struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    int64 `gorm:"uniqueIndex:idx_user_listing"`
	ListingID int64 `gorm:"uniqueIndex:idx_user_listing"`
}

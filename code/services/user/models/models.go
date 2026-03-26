package models

type User struct {
	ID       string `gorm:"primaryKey;type:uuid"`
	Name     string
	Email    string `gorm:"uniqueIndex"`
	Password string
}

type Favorite struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    string `gorm:"type:uuid;uniqueIndex:idx_user_listing"`
	ListingID string `gorm:"uniqueIndex:idx_user_listing"`
}

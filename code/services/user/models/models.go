package models

type User struct {
	ID          int64 `gorm:"primaryKey;autoIncrement"`
	Name        string
	Email       string `gorm:"uniqueIndex"`
	Password    string
}

type Favorite struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    int64 `gorm:"uniqueIndex:idx_user_listing"`
	ListingID int64 `gorm:"uniqueIndex:idx_user_listing"`
}

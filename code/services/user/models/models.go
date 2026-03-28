package models

type User struct {
	ID          string `gorm:"primaryKey;type:uuid"`
	Name        string
	Email       string `gorm:"uniqueIndex"`
	Password    string
	IsSeller    bool   `gorm:"default:false"`
	SellerType  string `gorm:"type:varchar(50)"`
	ContactInfo string
	Rating      float64
}

type Favorite struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    string `gorm:"type:uuid;uniqueIndex:idx_user_listing"`
	ListingID string `gorm:"uniqueIndex:idx_user_listing"`
}

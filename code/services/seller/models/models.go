package models

type Seller struct {
	ID          int64 `gorm:"primaryKey;autoIncrement"`
	Name        string
	SellerType  string `gorm:"type:varchar(50)"`
	ContactInfo string
	Rating      float64
}

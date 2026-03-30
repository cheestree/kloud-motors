package models

type Seller struct {
	ID          string `gorm:"primaryKey;type:uuid"`
	Name        string
	SellerType  string `gorm:"type:varchar(50)"`
	ContactInfo string
	Rating      float64
}

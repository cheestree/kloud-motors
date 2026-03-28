package models

type Seller struct {
	ID          string `gorm:"primaryKey;type:uuid"`
	Name        string
	IsSeller    bool   `gorm:"default:false"`
	SellerType  string `gorm:"type:varchar(50)"`
	ContactInfo string
	Rating      float64
}

func (Seller) TableName() string {
	return "users"
}

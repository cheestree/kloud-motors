package models

import "time"

type ListingBrand struct {
	ID   int64  `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (ListingBrand) TableName() string {
	return "brand"
}

type ListingModel struct {
	ID      int64  `gorm:"primaryKey;column:id"`
	BrandID int64  `gorm:"column:brand_id"`
	Name    string `gorm:"column:name"`
}

func (ListingModel) TableName() string {
	return "model"
}

type ListingFuelType struct {
	ID   int64  `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (ListingFuelType) TableName() string {
	return "fuel_type"
}

type ListingTransmission struct {
	ID   int64  `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (ListingTransmission) TableName() string {
	return "transmission"
}

type AutomotiveData struct {
	ID             int64     `gorm:"primaryKey;column:id"`
	Vin            string    `gorm:"column:vin"`
	AskPrice       *int64    `gorm:"column:ask_price"`
	Mileage        *int64    `gorm:"column:mileage"`
	ModelYear      *int32    `gorm:"column:model_year"`
	Trim           *string   `gorm:"column:trim"`
	City           *string   `gorm:"column:city"`
	District       *string   `gorm:"column:district"`
	State          *string   `gorm:"column:state"`
	Country        *string   `gorm:"column:country"`
	Color          *string   `gorm:"column:color"`
	DealerID       int64     `gorm:"column:dealer_id"`
	IsNew          bool      `gorm:"column:is_new"`
	BrandID        *int64    `gorm:"column:brand_id"`
	ModelID        *int64    `gorm:"column:model_id"`
	FuelTypeID     *int64    `gorm:"column:fuel_type_id"`
	TransmissionID *int64    `gorm:"column:transmission_id"`
	FirstSeen      time.Time `gorm:"column:first_seen"`
	LastSeen       time.Time `gorm:"column:last_seen"`
}

func (AutomotiveData) TableName() string {
	return "automotive_data"
}

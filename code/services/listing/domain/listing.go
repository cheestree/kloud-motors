package domain

type ListingDetails struct {
	ID           int64
	Make         string
	Model        string
	Year         int32
	Price        float64
	Mileage      int32
	Location     string
	FuelType     string
	Trim         string
	Transmission string
	Color        string
	SellerType   string
	Description  string
	ListedAt     string
	Images       []string
}

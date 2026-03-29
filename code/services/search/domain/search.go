package domain

type SearchParams struct {
	Make         string
	Model        string
	Year         int32
	MinPrice     int64
	MaxPrice     int64
	MaxMileage   int32
	FuelType     string
	BodyClass    string
	DriveType    string
	Transmission string
	IsNew        *bool
	Page         int32
	PageSize     int32
}

type ListingSummary struct {
	ID           int64
	Make         string
	Model        string
	Year         int32
	Price        int64
	Mileage      int32
	FuelType     string
	BodyClass    string
	DriveType    string
	Transmission string
	IsNew        bool
}

type SearchResult struct {
	Total    int32
	Page     int32
	PageSize int32
	Listings []ListingSummary
}

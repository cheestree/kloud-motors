package domain

import "services/shared"

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
	State        string
	District     string
	City         string
	Country      string
	IncludeSold  bool
}

type SearchResult struct {
	Total    int32
	Page     int32
	PageSize int32
	Listings []shared.ListingSummary
}

package listing

import (
	listingpb "services/listing/proto"
	searchpb "services/search/proto"
	"services/utils"
)

const (
	defaultListingPage     int32 = 1
	defaultListingPageSize int32 = 20
)

type ListingSearchQuery struct {
	Make        string `schema:"make" validate:"omitempty"`
	Model       string `schema:"model" validate:"omitempty"`
	Year        *int32 `schema:"year" validate:"omitempty,gte=1886"`
	MinPrice    *int64 `schema:"minPrice" validate:"omitempty,gte=0"`
	MaxPrice    *int64 `schema:"maxPrice" validate:"omitempty,gte=0"`
	MaxMileage  *int32 `schema:"maxMileage" validate:"omitempty,gte=0"`
	FuelType    string `schema:"fuelType" validate:"omitempty"`
	Page        int32  `schema:"page" validate:"gte=1"`
	PageSize    int32  `schema:"pageSize" validate:"gte=1,lte=100"`
	IncludeSold *bool  `schema:"includeSold" validate:"omitempty"`
}

type ListingListQuery struct {
	Page        int32 `schema:"page" validate:"gte=1"`
	PageSize    int32 `schema:"pageSize" validate:"gte=1,lte=100"`
	IncludeSold *bool `schema:"includeSold" validate:"omitempty"`
}

type ListingMutationBody struct {
	Vin          string  `json:"vin" validate:"notblank"`
	Make         string  `json:"make" validate:"notblank"`
	Model        string  `json:"model" validate:"notblank"`
	Year         int32   `json:"year" validate:"gte=1886,lte=2100"`
	Price        float64 `json:"price" validate:"gte=0"`
	Mileage      int32   `json:"mileage" validate:"gte=0"`
	City         string  `json:"city" validate:"omitempty"`
	District     string  `json:"district" validate:"omitempty"`
	State        string  `json:"state" validate:"omitempty"`
	Country      string  `json:"country" validate:"omitempty"`
	FuelType     string  `json:"fuel_type" validate:"omitempty"`
	BodyClass    string  `json:"body_class" validate:"omitempty"`
	DriveType    string  `json:"drive_type" validate:"omitempty"`
	Transmission string  `json:"transmission" validate:"omitempty"`
	Trim         string  `json:"trim" validate:"omitempty"`
	Color        string  `json:"color" validate:"omitempty"`
	IsNew        bool    `json:"is_new" validate:"omitempty"`
	IsSold       bool    `json:"is_sold" validate:"omitempty"`
}

type listingIDPath struct {
	ID int64 `schema:"id" validate:"gt=0"`
}

type ListingCompareQuery struct {
	IDs []int64 `schema:"ids" validate:"min=1,dive,gt=0"`
}

func DefaultListingListQuery() ListingListQuery {
	return ListingListQuery{
		Page:     defaultListingPage,
		PageSize: defaultListingPageSize,
	}
}

func DefaultListingSearchQuery() ListingSearchQuery {
	return ListingSearchQuery{
		Page:     defaultListingPage,
		PageSize: defaultListingPageSize,
	}
}

func (q ListingListQuery) SearchRequest() *searchpb.SearchRequest {
	return &searchpb.SearchRequest{
		Page:        q.Page,
		PageSize:    q.PageSize,
		IncludeSold: utils.BoolPtrToProtoBoolValue(q.IncludeSold),
	}
}

func (q ListingSearchQuery) SearchRequest() *searchpb.SearchRequest {
	return &searchpb.SearchRequest{
		Make:        q.Make,
		Model:       q.Model,
		Year:        utils.Int32ValueFromPtrOrZero(q.Year),
		MinPrice:    utils.Int64ValueFromPtrOrZero(q.MinPrice),
		MaxPrice:    utils.Int64ValueFromPtrOrZero(q.MaxPrice),
		MaxMileage:  utils.Int32ValueFromPtrOrZero(q.MaxMileage),
		FuelType:    q.FuelType,
		Page:        q.Page,
		PageSize:    q.PageSize,
		IncludeSold: utils.BoolPtrToProtoBoolValue(q.IncludeSold),
	}
}

func BuildCreateListingRequest(body ListingMutationBody, sellerID int64) *listingpb.CreateListingRequest {
	return &listingpb.CreateListingRequest{
		Vin:          body.Vin,
		Make:         body.Make,
		Model:        body.Model,
		Year:         body.Year,
		Price:        body.Price,
		Mileage:      body.Mileage,
		City:         body.City,
		District:     body.District,
		State:        body.State,
		Country:      body.Country,
		FuelType:     body.FuelType,
		BodyClass:    body.BodyClass,
		DriveType:    body.DriveType,
		Transmission: body.Transmission,
		Trim:         body.Trim,
		Color:        body.Color,
		IsNew:        body.IsNew,
		IsSold:       body.IsSold,
		SellerId:     sellerID,
	}
}

func BuildUpdateListingRequest(body ListingMutationBody, listingID, sellerID int64) *listingpb.UpdateListingRequest {
	return &listingpb.UpdateListingRequest{
		Id:           listingID,
		SellerId:     sellerID,
		Vin:          body.Vin,
		Make:         body.Make,
		Model:        body.Model,
		Year:         body.Year,
		Price:        body.Price,
		Mileage:      body.Mileage,
		City:         body.City,
		District:     body.District,
		State:        body.State,
		Country:      body.Country,
		FuelType:     body.FuelType,
		BodyClass:    body.BodyClass,
		DriveType:    body.DriveType,
		Transmission: body.Transmission,
		Trim:         body.Trim,
		Color:        body.Color,
		IsNew:        body.IsNew,
	}
}

func BuildListingDetailsRequest(id int64) *listingpb.ListingDetailsRequest {
	return &listingpb.ListingDetailsRequest{Id: id}
}

func BuildDeleteListingRequest(id, sellerID int64) *listingpb.DeleteListingRequest {
	return &listingpb.DeleteListingRequest{
		Id:       id,
		SellerId: sellerID,
	}
}

func BuildCompareListingsRequest(ids []int64) *listingpb.CompareListingsRequest {
	return &listingpb.CompareListingsRequest{Ids: ids}
}

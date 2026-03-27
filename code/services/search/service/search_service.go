package service

import (
	"context"
	"search/repository"
)

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

type SearchResult struct {
	Total    int32
	Page     int32
	PageSize int32
	Listings []repository.ListingSummary
}

type SearchService struct {
	repository *repository.SearchRepository
}

func NewSearchService(repository *repository.SearchRepository) *SearchService {
	return &SearchService{repository: repository}
}

func (s *SearchService) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filters := repository.SearchFilters{
		Make:         params.Make,
		Model:        params.Model,
		Year:         params.Year,
		MinPrice:     params.MinPrice,
		MaxPrice:     params.MaxPrice,
		MaxMileage:   params.MaxMileage,
		FuelType:     params.FuelType,
		BodyClass:    params.BodyClass,
		DriveType:    params.DriveType,
		Transmission: params.Transmission,
		IsNew:        params.IsNew,
		Page:         page,
		PageSize:     pageSize,
	}

	listings, total, err := s.repository.Search(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Listings: listings,
	}, nil
}

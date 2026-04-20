package service

import (
	"context"
	"os"
	"services/search/domain"
	"strconv"
	"strings"

	"services/search/repository"
)

type SearchService struct {
	repository      *repository.SearchRepository
	defaultPage     int32
	defaultPageSize int32
	maxPageSize     int32
}

const (
	envSearchDefaultPage     = "SEARCH_DEFAULT_PAGE"
	envSearchDefaultPageSize = "SEARCH_DEFAULT_PAGE_SIZE"
	envSearchMaxPageSize     = "SEARCH_MAX_PAGE_SIZE"

	fallbackDefaultPage     int32 = 1
	fallbackDefaultPageSize int32 = 20
	fallbackMaxPageSize     int32 = 100
)

func NewSearchService(repository *repository.SearchRepository) *SearchService {
	return &SearchService{
		repository:      repository,
		defaultPage:     getEnvInt32(envSearchDefaultPage, fallbackDefaultPage),
		defaultPageSize: getEnvInt32(envSearchDefaultPageSize, fallbackDefaultPageSize),
		maxPageSize:     getEnvInt32(envSearchMaxPageSize, fallbackMaxPageSize),
	}
}

func (s *SearchService) Search(ctx context.Context, params domain.SearchParams) (*domain.SearchResult, error) {
	page := params.Page
	if page <= 0 {
		page = s.defaultPage
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = s.defaultPageSize
	}
	if pageSize > s.maxPageSize {
		pageSize = s.maxPageSize
	}

	filters := domain.SearchParams{
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
		State:        params.State,
		District:     params.District,
		City:         params.City,
		Country:      params.Country,
		IncludeSold:  params.IncludeSold,
	}

	listings, total, err := s.repository.Search(ctx, filters)
	if err != nil {
		return nil, err
	}

	return &domain.SearchResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Listings: listings,
	}, nil
}

func getEnvInt32(key string, fallback int32) int32 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return int32(parsed)
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"services/search/domain"
	"services/search/repository"
	"services/shared/cache"
	"services/utils"
)

type SearchService struct {
	repository      *repository.SearchRepository
	redisCache      *cache.RedisCache
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

func NewSearchService(repository *repository.SearchRepository, redisCache *cache.RedisCache) *SearchService {
	return &SearchService{
		repository:      repository,
		redisCache:      redisCache,
		defaultPage:     utils.GetEnvInt32(envSearchDefaultPage, fallbackDefaultPage),
		defaultPageSize: utils.GetEnvInt32(envSearchDefaultPageSize, fallbackDefaultPageSize),
		maxPageSize:     utils.GetEnvInt32(envSearchMaxPageSize, fallbackMaxPageSize),
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

	var cacheKey string
	if s.redisCache != nil {
		filtersBytes, _ := json.Marshal(filters)
		hash := sha256.Sum256(filtersBytes)
		cacheKey = "search:query:" + hex.EncodeToString(hash[:])

		var cachedResult domain.SearchResult
		if err := s.redisCache.Get(ctx, cacheKey, &cachedResult); err == nil {
			return &cachedResult, nil
		}
	}

	listings, total, err := s.repository.Search(ctx, filters)
	if err != nil {
		return nil, err
	}

	result := &domain.SearchResult{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Listings: listings,
	}

	if s.redisCache != nil {
		_ = s.redisCache.Set(ctx, cacheKey, result)
	}

	return result, nil
}

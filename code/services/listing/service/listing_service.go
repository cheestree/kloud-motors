package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"services/listing/repository"
	"services/shared"
	"services/shared/cache"
)

type ListingService struct {
	repository *repository.ListingRepository
	redisCache *cache.RedisCache
}

func NewListingService(repository *repository.ListingRepository, redisCache *cache.RedisCache) *ListingService {
	return &ListingService{repository: repository, redisCache: redisCache}
}

func (s *ListingService) invalidateListingCache(ctx context.Context, id int64) {
	if s.redisCache != nil {
		_ = s.redisCache.Delete(ctx, fmt.Sprintf("listing:details:%d", id))
		_ = s.redisCache.Delete(ctx, fmt.Sprintf("listing:summary:%d", id))
	}
}

func (s *ListingService) GetListingDetails(ctx context.Context, id int64) (*shared.ListingDetails, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be a positive integer")
	}

	var listing shared.ListingDetails
	cacheKey := fmt.Sprintf("listing:details:%d", id)
	if s.redisCache != nil {
		err := s.redisCache.Get(ctx, cacheKey, &listing)
		if err == nil {
			return &listing, nil // Cache hit
		}
	}

	repoListing, err := s.repository.GetListingDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	if repoListing == nil {
		return nil, ErrListingNotFound
	}

	if s.redisCache != nil {
		_ = s.redisCache.Set(ctx, cacheKey, repoListing)
	}

	return repoListing, nil
}

func (s *ListingService) CreateListing(ctx context.Context, listing repository.ListingMutation) (*shared.ListingDetails, error) {
	if err := validateListingMutation(listing); err != nil {
		return nil, err
	}
	created, err := s.repository.CreateListing(ctx, listing)
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, ErrListingNotFound
	}
	return created, nil
}

func (s *ListingService) UpdateListing(ctx context.Context, id int64, listing repository.ListingMutation) (*shared.ListingDetails, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be a positive integer")
	}
	if err := validateListingMutation(listing); err != nil {
		return nil, err
	}
	// Prevent updating is_sold via update
	listing.IsSold = false // or zero value, will be ignored in repo
	updated, err := s.repository.UpdateListing(ctx, id, listing)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrListingNotFound
	}
	s.invalidateListingCache(ctx, id)
	return updated, nil
}

func (s *ListingService) SetListingSoldStatus(ctx context.Context, id int64, dealerID int64, isSold bool) (*shared.ListingDetails, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be a positive integer")
	}
	if dealerID <= 0 {
		return nil, fmt.Errorf("invalid dealer_id: must be a positive integer")
	}
	updated, err := s.repository.SetListingSoldStatus(ctx, id, dealerID, isSold)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrListingNotFound
	}
	s.invalidateListingCache(ctx, id)
	return updated, nil
}

func (s *ListingService) DeleteListing(ctx context.Context, id int64, dealerID int64) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("invalid ID: must be a positive integer")
	}
	if dealerID <= 0 {
		return false, fmt.Errorf("invalid dealer_id: must be a positive integer")
	}
	deleted, err := s.repository.DeleteListing(ctx, id, dealerID)
	if err != nil {
		return false, err
	}
	if !deleted {
		return false, ErrListingNotFound
	}
	s.invalidateListingCache(ctx, id)
	return true, nil
}

func (s *ListingService) CheckListingOwnership(ctx context.Context, listingID int64, dealerID int64) (bool, error) {
	if listingID <= 0 {
		return false, fmt.Errorf("invalid listing_id: must be a positive integer")
	}
	if dealerID <= 0 {
		return false, fmt.Errorf("invalid dealer_id: must be a positive integer")
	}
	return s.repository.CheckListingOwnership(ctx, listingID, dealerID)
}

func (s *ListingService) CompareListings(ctx context.Context, ids []int64) ([]*shared.ListingDetails, error) {
	if len(ids) == 0 {
		return []*shared.ListingDetails{}, nil
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("invalid ID: must be a positive integer")
		}
	}
	listings, err := s.repository.CompareListings(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(listings) != len(ids) {
		return nil, ErrListingNotFound
	}
	return listings, nil
}

func (s *ListingService) GetListingSummary(ctx context.Context, id int64) (*shared.ListingSummary, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be a positive integer")
	}

	var summary shared.ListingSummary
	cacheKey := fmt.Sprintf("listing:summary:%d", id)
	if s.redisCache != nil {
		err := s.redisCache.Get(ctx, cacheKey, &summary)
		if err == nil {
			return &summary, nil // Cache hit
		}
	}

	repoSummary, err := s.repository.GetListingSummary(ctx, id)
	if err != nil {
		return nil, err
	}
	if repoSummary == nil {
		return nil, ErrListingNotFound
	}

	if s.redisCache != nil {
		_ = s.redisCache.Set(ctx, cacheKey, repoSummary)
	}

	return repoSummary, nil
}

func (s *ListingService) GetListingSummaries(ctx context.Context, ids []int64) ([]*shared.ListingSummary, error) {
	if len(ids) == 0 {
		return []*shared.ListingSummary{}, nil
	}
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("invalid ID: must be a positive integer")
		}
	}

	listings, err := s.repository.GetListingSummaries(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(listings) != len(ids) {
		return nil, ErrListingNotFound
	}

	return listings, nil
}

func (s *ListingService) CheckListingOpen(ctx context.Context, id int64) (bool, int64, error) {
	if id <= 0 {
		return false, 0, fmt.Errorf("invalid ID: must be a positive integer")
	}
	open, dealerID, err := s.repository.CheckListingOpen(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, ErrListingNotFound
		}
		return false, 0, err
	}
	return open, dealerID, nil
}

func validateListingMutation(listing repository.ListingMutation) error {
	if strings.TrimSpace(listing.Vin) == "" {
		return fmt.Errorf("vin is required")
	}
	if strings.TrimSpace(listing.Make) == "" {
		return fmt.Errorf("make is required")
	}
	if strings.TrimSpace(listing.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if listing.DealerID <= 0 {
		return fmt.Errorf("dealer_id must be a positive integer")
	}
	if listing.Year <= 0 {
		return fmt.Errorf("year must be a positive integer")
	}
	if listing.Year < 1886 || listing.Year > 2100 {
		return fmt.Errorf("year must be between 1886 and 2100")
	}
	if listing.Price < 0 {
		return fmt.Errorf("price cannot be negative")
	}
	if listing.Mileage < 0 {
		return fmt.Errorf("mileage cannot be negative")
	}
	return nil
}

var ErrListingNotFound = fmt.Errorf("listing not found")

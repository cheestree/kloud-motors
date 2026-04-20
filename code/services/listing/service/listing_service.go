package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"services/listing/repository"
	"services/shared"
)

type ListingService struct {
	repository *repository.ListingRepository
}

func NewListingService(repository *repository.ListingRepository) *ListingService {
	return &ListingService{repository: repository}
}

func (s *ListingService) GetListingDetails(ctx context.Context, id int64) (*shared.ListingDetails, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid ID: must be a positive integer")
	}
	listing, err := s.repository.GetListingDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, ErrListingNotFound
	}
	return listing, nil
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
	summary, err := s.repository.GetListingSummary(ctx, id)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, ErrListingNotFound
	}
	return summary, nil
}

func (s *ListingService) CheckListingOpen(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, fmt.Errorf("invalid ID: must be a positive integer")
	}
	open, err := s.repository.CheckListingOpen(ctx, id)
	if err != nil {
		return false, err
	}
	if !open {
		return false, ErrListingNotFound
	}
	return open, nil
}

var ErrListingNotFound = fmt.Errorf("listing not found")

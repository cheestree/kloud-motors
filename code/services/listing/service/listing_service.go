package service

import (
	"context"
	"fmt"
	"listing/domain"
	"listing/repository"
)

type ListingService struct {
	repository *repository.ListingRepository
}

func NewListingService(repository *repository.ListingRepository) *ListingService {
	return &ListingService{repository: repository}
}

func (s *ListingService) GetListingDetails(ctx context.Context, id int64) (*domain.ListingDetails, error) {
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

func (s *ListingService) CompareListings(ctx context.Context, ids []int64) ([]*domain.ListingDetails, error) {
	if len(ids) == 0 {
		return []*domain.ListingDetails{}, nil
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

func (s *ListingService) CheckListingOwnership(ctx context.Context, listingID int64, dealerID int64) (bool, error) {
	if listingID <= 0 {
		return false, fmt.Errorf("invalid listing_id: must be a positive integer")
	}
	if dealerID <= 0 {
		return false, fmt.Errorf("invalid dealer_id: must be a positive integer")
	}
	open, err := s.repository.CheckListingOpen(ctx, listingID)
	if err != nil {
		return false, err
	}
	if !open {
		return false, ErrListingNotFound
	}
	return s.repository.CheckListingOwnership(ctx, listingID, dealerID)
}

func (s *ListingService) CheckListingOpen(ctx context.Context, listingID int64) (bool, error) {
	if listingID <= 0 {
		return false, fmt.Errorf("invalid ID: must be a positive integer")
	}
	return s.repository.CheckListingOpen(ctx, listingID)
}

var ErrListingNotFound = fmt.Errorf("listing not found")

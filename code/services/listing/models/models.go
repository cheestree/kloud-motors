package models

import (
	"errors"
	"services/listing/proto"
	"services/listing/service"
	"services/shared"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ToListingDetailsResponse(listing *shared.ListingDetails) *proto.ListingDetailsResponse {
	if listing == nil {
		return nil
	}
	return &proto.ListingDetailsResponse{
		Id:           listing.Id,
		Make:         listing.Make,
		Model:        listing.Model,
		Year:         listing.Year,
		Price:        float64(listing.Price),
		Mileage:      listing.Mileage,
		City:         listing.City,
		District:     listing.District,
		State:        listing.State,
		Country:      listing.Country,
		FuelType:     listing.FuelType,
		Trim:         listing.Trim,
		Transmission: listing.Transmission,
		Color:        listing.Color,
		SellerType:   listing.SellerType,
		Description:  listing.Description,
		ListedAt:     listing.LastSeen,
		Images:       listing.Images,
		IsSold:       listing.IsSold,
	}
}

func ToListingSummary(summary *shared.ListingSummary) *shared.ListingSummary {
	if summary == nil {
		return nil
	}

	return &shared.ListingSummary{
		Id:           summary.Id,
		DealerId:     summary.DealerId,
		Make:         summary.Make,
		Model:        summary.Model,
		Year:         summary.Year,
		Price:        summary.Price,
		Mileage:      summary.Mileage,
		FuelType:     summary.FuelType,
		BodyClass:    summary.BodyClass,
		DriveType:    summary.DriveType,
		Transmission: summary.Transmission,
		IsNew:        summary.IsNew,
		City:         summary.City,
		District:     summary.District,
		State:        summary.State,
		Country:      summary.Country,
		LastSeen:     summary.LastSeen,
		IsSold:       summary.IsSold,
	}
}

func MapListingError(action string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, service.ErrListingNotFound) {
		return status.Error(codes.NotFound, "listing not found")
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "required") ||
		strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "must be") ||
		strings.Contains(lower, "cannot") ||
		strings.Contains(lower, "unknown") {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique") {
		return status.Error(codes.AlreadyExists, "listing with this VIN already exists")
	}

	return status.Errorf(codes.Internal, "failed to %s: %v", action, err)
}

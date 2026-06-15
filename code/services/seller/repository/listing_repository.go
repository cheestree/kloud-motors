package repository

import (
	"context"
	"errors"
	"time"

	. "services/seller/models"
	proto "services/seller/proto"
	"services/utils"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	ListingDB *gorm.DB
	SellerDB  *gorm.DB
}

func NewRepository(listingDB *gorm.DB, sellerDB *gorm.DB) *Repository {
	return &Repository{ListingDB: listingDB, SellerDB: sellerDB}
}

func getOrCreateLookup(ctx context.Context, tx *gorm.DB, kind string, name string, brandID *int64) (*int64, error) {
	if name == "" {
		return nil, nil
	}

	switch kind {
	case "brand":
		var brand ListingBrand
		err := tx.WithContext(ctx).Where("name = ?", name).First(&brand).Error
		if err == nil {
			return &brand.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		brand = ListingBrand{Name: name}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&brand).Error; err != nil {
			return nil, err
		}
		if brand.ID == 0 {
			if err := tx.WithContext(ctx).Where("name = ?", name).First(&brand).Error; err != nil {
				return nil, err
			}
		}
		return &brand.ID, nil
	case "model":
		if brandID == nil {
			return nil, nil
		}
		var model ListingModel
		err := tx.WithContext(ctx).Where("brand_id = ? AND name = ?", *brandID, name).First(&model).Error
		if err == nil {
			return &model.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		model = ListingModel{BrandID: *brandID, Name: name}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "brand_id"}, {Name: "name"}},
			DoNothing: true,
		}).Create(&model).Error; err != nil {
			return nil, err
		}
		if model.ID == 0 {
			if err := tx.WithContext(ctx).Where("brand_id = ? AND name = ?", *brandID, name).First(&model).Error; err != nil {
				return nil, err
			}
		}
		return &model.ID, nil
	case "fuel_type":
		var fuelType ListingFuelType
		err := tx.WithContext(ctx).Where("name = ?", name).First(&fuelType).Error
		if err == nil {
			return &fuelType.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		fuelType = ListingFuelType{Name: name}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&fuelType).Error; err != nil {
			return nil, err
		}
		if fuelType.ID == 0 {
			if err := tx.WithContext(ctx).Where("name = ?", name).First(&fuelType).Error; err != nil {
				return nil, err
			}
		}
		return &fuelType.ID, nil
	case "transmission":
		var transmission ListingTransmission
		err := tx.WithContext(ctx).Where("name = ?", name).First(&transmission).Error
		if err == nil {
			return &transmission.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		transmission = ListingTransmission{Name: name}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&transmission).Error; err != nil {
			return nil, err
		}
		if transmission.ID == 0 {
			if err := tx.WithContext(ctx).Where("name = ?", name).First(&transmission).Error; err != nil {
				return nil, err
			}
		}
		return &transmission.ID, nil
	default:
		return nil, errors.New("unsupported lookup type")
	}
}

func CreateListing(ctx context.Context, listingDB *gorm.DB, req *proto.CreateListingRequest) (int64, time.Time, error) {
	var listingID int64
	var listedAt time.Time
	err := listingDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		brandID, err := getOrCreateLookup(ctx, tx, "brand", req.Make, nil)
		if err != nil {
			return err
		}
		modelID, err := getOrCreateLookup(ctx, tx, "model", req.Model, brandID)
		if err != nil {
			return err
		}
		fuelTypeID, err := getOrCreateLookup(ctx, tx, "fuel_type", req.FuelType, nil)
		if err != nil {
			return err
		}
		transmissionID, err := getOrCreateLookup(ctx, tx, "transmission", req.Transmission, nil)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		yearValue := utils.PositiveInt32Ptr(req.Year)
		priceValue := utils.PositiveInt64PtrFromFloat(req.Price)
		mileageValue := utils.PositiveInt64PtrFromInt32(req.Mileage)
		trimValue := utils.StringPtrIfNotEmpty(req.Trim)
		cityValue := utils.StringPtrIfNotEmpty(req.City)
		districtValue := utils.StringPtrIfNotEmpty(req.District)
		stateValue := utils.StringPtrIfNotEmpty(req.State)
		countryValue := utils.StringPtrIfNotEmpty(req.Country)
		colorValue := utils.StringPtrIfNotEmpty(req.Color)

		listing := AutomotiveData{
			Vin:            req.Vin,
			AskPrice:       priceValue,
			Mileage:        mileageValue,
			ModelYear:      yearValue,
			Trim:           trimValue,
			City:           cityValue,
			District:       districtValue,
			State:          stateValue,
			Country:        countryValue,
			Color:          colorValue,
			DealerID:       req.DealerId,
			IsNew:          req.IsNew,
			BrandID:        brandID,
			ModelID:        modelID,
			FuelTypeID:     fuelTypeID,
			TransmissionID: transmissionID,
			FirstSeen:      now,
			LastSeen:       now,
		}

		if err := tx.WithContext(ctx).Create(&listing).Error; err != nil {
			return err
		}

		listingID = listing.ID
		listedAt = listing.FirstSeen
		return nil
	})
	if err != nil {
		return 0, time.Time{}, err
	}

	return listingID, listedAt, nil
}

func GetSellersPreview(ctx context.Context, db *gorm.DB, sellerIDs []int64) ([]*proto.SellerPreview, error) {
	var sellers []Seller
	if err := db.Where("id IN ?", sellerIDs).Find(&sellers).Error; err != nil {
		return nil, status.Error(codes.Internal, "failed to get sellers preview")
	}

	previews := make([]*proto.SellerPreview, 0, len(sellers))
	for _, seller := range sellers {
		previews = append(previews, &proto.SellerPreview{
			Id:   seller.ID,
			Name: seller.Name,
		})
	}

	return previews, nil
}

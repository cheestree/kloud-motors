package main

import (
    "context"
    "errors"
    "log"
    "os"
    "time"

    . "seller/models"
    proto "seller/proto"

    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

func initListingDB() {
	dsn := os.Getenv("LISTING_DATABASE_URL")
	if dsn == "" {
		log.Fatalf("LISTING_DATABASE_URL is not set")
	}

	var err error
	listingDb, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect listing database: %v", err)
	}
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

func createListing(ctx context.Context, req *proto.CreateListingRequest) (int64, time.Time, error) {
	var listingID int64
	var listedAt time.Time
	err := listingDb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		yearValue := int32Ptr(req.Year)
		priceValue := int64PtrFromFloat(req.Price)
		mileageValue := int64PtrFromInt32(req.Mileage)
		trimValue := stringPtr(req.Trim)
		cityValue := stringPtr(req.City)
		districtValue := stringPtr(req.District)
		stateValue := stringPtr(req.State)
		countryValue := stringPtr(req.Country)
		colorValue := stringPtr(req.Color)

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

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func int32Ptr(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	return &value
}

func int64PtrFromInt32(value int32) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func int64PtrFromFloat(value float64) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}
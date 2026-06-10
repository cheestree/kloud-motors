package marketprice

import (
	marketpricepb "services/marketprice/proto"
	"services/utils"
)

type AveragePriceQuery struct {
	Brand    string `schema:"brand" validate:"notblank"`
	Model    string `schema:"model" validate:"notblank"`
	YearFrom *int32 `schema:"year_from" validate:"omitempty,gte=0"`
	YearTo   *int32 `schema:"year_to" validate:"omitempty,gte=0"`
}

func BuildAveragePriceRequest(query AveragePriceQuery) *marketpricepb.AveragePriceRequest {
	return &marketpricepb.AveragePriceRequest{
		Brand:    query.Brand,
		Model:    query.Model,
		YearFrom: utils.Int32ValueFromPtrOrZero(query.YearFrom),
		YearTo:   utils.Int32ValueFromPtrOrZero(query.YearTo),
	}
}

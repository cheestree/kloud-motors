package geo

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
)

var (
	queryDecoder = schema.NewDecoder()
	validatorV10 = validator.New()
)

func init() {
	queryDecoder.IgnoreUnknownKeys(true)
	validatorV10.RegisterValidation("notblank", func(field validator.FieldLevel) bool {
		return strings.TrimSpace(field.Field().String()) != ""
	})
	validatorV10.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, tag := range []string{"schema", "json"} {
			name := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return ""
	})
	validatorV10.RegisterStructValidation(validateAggregatesYearRange, AggregatesQuery{})
	validatorV10.RegisterStructValidation(validateByLocationYearRange, ByLocationQuery{})
}

func BindAndValidateQuery(r *http.Request, target interface{}) error {
	if err := queryDecoder.Decode(target, r.URL.Query()); err != nil {
		return err
	}
	return Validate(target)
}

func Validate(target interface{}) error {
	return validatorV10.Struct(target)
}

func validateAggregatesYearRange(sl validator.StructLevel) {
	query := sl.Current().Interface().(AggregatesQuery)
	reportInvalidYearRange(sl, query.YearFrom, query.YearTo)
}

func validateByLocationYearRange(sl validator.StructLevel) {
	query := sl.Current().Interface().(ByLocationQuery)
	reportInvalidYearRange(sl, query.YearFrom, query.YearTo)
}

func reportInvalidYearRange(sl validator.StructLevel, yearFrom, yearTo *int32) {
	if yearFrom == nil || yearTo == nil || *yearTo >= *yearFrom {
		return
	}
	sl.ReportError(*yearTo, "year_to", "year_to", "gtefield", "year_from")
}

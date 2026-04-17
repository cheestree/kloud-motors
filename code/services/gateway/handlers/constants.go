package handlers

const (
	headerContentType = "Content-Type"
	headerAuth        = "Authorization"

	contentTypeJSON = "application/json"
)

const (
	msgMethodNotAllowed = "Method not allowed"
	msgInvalidBody      = "Invalid request body"
	msgUnauthorized     = "Unauthorized"
	msgNotFound         = "Not found"
)

const (
	errMissingAuthHeader = "missing authorization header"
	errInvalidAuthHeader = "invalid authorization header format"
	errJWTNotConfigured  = "jwt secret not configured"
	errInvalidToken      = "invalid token"
	errUserIDNotInToken  = "user id not found in token"
)

const (
	authSchemeBearer = "Bearer"
	envJWTSecret     = "JWT_SECRET"
)

const (
	pathActionBid  = "bid"
	pathActionBids = "bids"
)

const (
	queryStatus   = "status"
	queryPage     = "page"
	queryPageSize = "page_size"

	queryMetrics   = "metrics"
	queryGroupBy   = "group_by"
	queryLocations = "locations"
	queryYearFrom  = "year_from"
	queryYearTo    = "year_to"
	queryLimit     = "limit"
	querySkip      = "skip"
	queryFuelType  = "fuel_type"
	querySortBy    = "sort_by"
	queryOrder     = "order"
	queryLocation  = "location"
	queryBrand     = "brand"
	queryModel     = "model"

	queryMake       = "make"
	queryYear       = "year"
	queryMinPrice   = "minPrice"
	queryMaxPrice   = "maxPrice"
	queryMaxMileage = "maxMileage"
	queryFuelTypeV2 = "fuelType"
	queryPageSizeV2 = "pageSize"
	queryIDs        = "ids"
)

const (
	groupByDistrict = "district"
	groupByCity     = "city"
	groupByCountry  = "country"
)

const (
	metricAvgPrice    = "avg_price"
	metricMedianPrice = "median_price"
	metricCount       = "count"
)

const (
	orderAsc  = "asc"
	orderDesc = "desc"
)

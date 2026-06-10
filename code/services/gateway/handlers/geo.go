package handlers

import (
	"net/http"

	georequests "services/gateway/handlers/geo"
)

func HandleMarketAggregates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	query := georequests.AggregatesQuery{}
	if err := georequests.BindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid aggregate filters: brand and model are required; metrics must be avg_price, median_price, or count; group_by must be district, city, or country; year_from and year_to must be at least 0 with year_to >= year_from; limit must be at least 1; skip must be at least 0", err)
		return
	}
	ctx := r.Context()

	resp, err := geoClient.Aggregates(ctx, georequests.BuildAggregatesRequest(query))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleMarketPriceComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	query := georequests.PriceComparisonQuery{}
	if err := georequests.BindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid price comparison filters: brand and model are required; group_by must be district, city, or country; sort_by must be avg_price or count; order must be asc or desc; limit must be at least 1; skip must be at least 0", err)
		return
	}
	ctx := r.Context()

	resp, err := geoClient.PriceComparison(ctx, georequests.BuildPriceComparisonRequest(query))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func HandleStatsByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	query := georequests.ByLocationQuery{}
	if err := georequests.BindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid location statistics filters: brand and model are required; year_from and year_to must be at least 0 with year_to >= year_from", err)
		return
	}
	ctx := r.Context()

	resp, err := geoClient.ByLocation(ctx, georequests.BuildByLocationRequest(query))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

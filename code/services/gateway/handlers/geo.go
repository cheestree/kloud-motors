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
		writeRequestError(w, "Invalid query parameters", err)
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
		writeRequestError(w, "Invalid query parameters", err)
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
		writeRequestError(w, "Invalid query parameters", err)
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

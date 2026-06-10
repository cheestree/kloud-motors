package handlers

import (
	"net/http"

	marketpricerequests "services/gateway/handlers/marketprice"
)

func HandleAveragePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, msgMethodNotAllowed, nil)
		return
	}

	query := marketpricerequests.AveragePriceQuery{}
	if err := marketpricerequests.BindAndValidateQuery(r, &query); err != nil {
		writeRequestError(w, "Invalid average price filters: brand and model are required; year_from and year_to must be at least 0 with year_to >= year_from", err)
		return
	}
	ctx := r.Context()

	resp, err := marketpriceClient.GetAverageMarketPrice(ctx, marketpricerequests.BuildAveragePriceRequest(query))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

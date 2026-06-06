package handlers

import (
	"net/http"

	marketpricepb "services/marketprice/proto"
	"services/utils"
)

func HandleAveragePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	ctx := r.Context()

	req := &marketpricepb.AveragePriceRequest{
		Brand: q.Get(queryBrand), // query param: "brand"
		Model: q.Get(queryModel), // query param: "model"
	}
	if s := q.Get(queryYearFrom); s != "" {
		req.YearFrom = utils.ParseInt32OrZero(s) // query param: "year_from"
	}
	if s := q.Get(queryYearTo); s != "" {
		req.YearTo = utils.ParseInt32OrZero(s) // query param: "year_to"
	}

	resp, err := marketpriceClient.GetAverageMarketPrice(ctx, req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

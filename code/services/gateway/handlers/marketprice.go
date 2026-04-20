package handlers

import (
	"context"
	"net/http"

	marketpricepb "services/marketprice/proto"
)


func HandleAveragePrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	ctx := context.Background()

	req := &marketpricepb.AveragePriceRequest{
		Brand: q.Get(queryBrand), // query param: "brand"
		Model: q.Get(queryModel), // query param: "model"
	}
	if s := q.Get(queryYearFrom); s != "" {
		req.YearFrom = parseInt32(s) // query param: "year_from"
	}
	if s := q.Get(queryYearTo); s != "" {
		req.YearTo = parseInt32(s) // query param: "year_to"
	}

	resp, err := marketpriceClient.GetAverageMarketPrice(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

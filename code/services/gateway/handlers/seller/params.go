package seller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

var ErrMissingSellerID = errors.New("missing seller id")

type sellerIDPath struct {
	ID int64 `schema:"seller_id" validate:"gt=0"`
}

func SellerIDFromPath(r *http.Request) (int64, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		return 0, ErrMissingSellerID
	}

	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return 0, err
	}

	path := sellerIDPath{ID: id}
	if err := Validate(path); err != nil {
		return 0, err
	}
	return id, nil
}

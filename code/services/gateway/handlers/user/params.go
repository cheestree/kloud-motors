package user

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

var ErrMissingListingID = errors.New("missing listing id")

type favoriteListingIDPath struct {
	ID int64 `schema:"listing_id" validate:"gt=0"`
}

func FavoriteListingIDFromPath(r *http.Request) (int64, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 6 || parts[5] == "" {
		return 0, ErrMissingListingID
	}

	id, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		return 0, err
	}

	path := favoriteListingIDPath{ID: id}
	if err := Validate(path); err != nil {
		return 0, err
	}
	return id, nil
}

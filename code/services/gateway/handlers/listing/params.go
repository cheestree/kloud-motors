package listing

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

var ErrMissingListingID = errors.New("missing listing id")

func ListingIDFromPath(r *http.Request) (int64, error) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 || parts[3] == "" {
		return 0, ErrMissingListingID
	}

	id, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return 0, err
	}

	path := listingIDPath{ID: id}
	if err := Validate(path); err != nil {
		return 0, err
	}
	return id, nil
}

func ParseCommaSeparatedInt64s(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	idStrs := strings.Split(raw, ",")
	ids := make([]int64, 0, len(idStrs))
	for _, s := range idStrs {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			return nil, strconv.ErrSyntax
		}
		id, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

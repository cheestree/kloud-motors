package service

import (
	"context"
	"testing"

	"services/search/domain"
	"services/shared"
)

type fakeSearcher struct {
	calls       int
	lastFilters domain.SearchParams
	listings    []shared.ListingSummary
	total       int32
	err         error
}

func (f *fakeSearcher) Search(_ context.Context, filters domain.SearchParams) ([]shared.ListingSummary, int32, error) {
	f.calls++
	f.lastFilters = filters
	return f.listings, f.total, f.err
}

func TestSearchService_DefaultPaging(t *testing.T) {
	t.Setenv("SEARCH_DEFAULT_PAGE", "2")
	t.Setenv("SEARCH_DEFAULT_PAGE_SIZE", "15")
	t.Setenv("SEARCH_MAX_PAGE_SIZE", "50")

	repo := &fakeSearcher{
		listings: []shared.ListingSummary{{Id: 1}},
		total:    1,
	}

	svc := NewSearchService(repo, nil)

	res, err := svc.Search(context.Background(), domain.SearchParams{
		Make:        "Ford",
		Page:        0,
		PageSize:    0,
		IncludeSold: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.calls != 1 {
		t.Fatalf("expected repository to be called once, got %d", repo.calls)
	}
	if repo.lastFilters.Page != 2 {
		t.Fatalf("expected page 2, got %d", repo.lastFilters.Page)
	}
	if repo.lastFilters.PageSize != 15 {
		t.Fatalf("expected page size 15, got %d", repo.lastFilters.PageSize)
	}
	if !repo.lastFilters.IncludeSold {
		t.Fatalf("expected include sold to be true")
	}
	if repo.lastFilters.Make != "Ford" {
		t.Fatalf("expected make Ford, got %q", repo.lastFilters.Make)
	}

	if res.Page != 2 {
		t.Fatalf("expected response page 2, got %d", res.Page)
	}
	if res.PageSize != 15 {
		t.Fatalf("expected response page size 15, got %d", res.PageSize)
	}
	if res.Total != 1 {
		t.Fatalf("expected total 1, got %d", res.Total)
	}
	if len(res.Listings) != 1 {
		t.Fatalf("expected 1 listing, got %d", len(res.Listings))
	}
}

func TestSearchService_MaxPageSize(t *testing.T) {
	t.Setenv("SEARCH_DEFAULT_PAGE", "1")
	t.Setenv("SEARCH_DEFAULT_PAGE_SIZE", "10")
	t.Setenv("SEARCH_MAX_PAGE_SIZE", "25")

	repo := &fakeSearcher{}
	svc := NewSearchService(repo, nil)

	_, err := svc.Search(context.Background(), domain.SearchParams{
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastFilters.PageSize != 25 {
		t.Fatalf("expected page size 25, got %d", repo.lastFilters.PageSize)
	}
}

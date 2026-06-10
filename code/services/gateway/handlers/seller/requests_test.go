package seller

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestSellerIDFromPathParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/sellers/7514", nil)

	sellerID, err := SellerIDFromPath(req)
	if err != nil {
		t.Fatalf("SellerIDFromPath returned error: %v", err)
	}

	protoReq := BuildGetSellerProfileRequest(sellerID)
	if protoReq.SellerId != 7514 {
		t.Fatalf("SellerId = %d, want 7514", protoReq.SellerId)
	}
}

func TestSellerIDFromPathRejectsMissingID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/sellers/", nil)

	_, err := SellerIDFromPath(req)
	if !errors.Is(err, ErrMissingSellerID) {
		t.Fatalf("error = %v, want ErrMissingSellerID", err)
	}
}

func TestSellerIDFromPathRejectsInvalidID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/sellers/not-an-id", nil)

	_, err := SellerIDFromPath(req)
	if err == nil {
		t.Fatal("SellerIDFromPath returned nil, want parse error")
	}
}

func TestSellerIDFromPathRejectsNonPositiveID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/sellers/0", nil)

	_, err := SellerIDFromPath(req)
	if err == nil {
		t.Fatal("SellerIDFromPath returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if validationErrs[0].Field() != "seller_id" || validationErrs[0].Tag() != "gt" {
		t.Fatalf("validation error = field %q tag %q, want seller_id gt", validationErrs[0].Field(), validationErrs[0].Tag())
	}
}

func TestBindAndValidateSellersPreviewBodyParsesAndBuildsRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/sellers/preview", strings.NewReader(`{"seller_ids":[7514,7515]}`))
	body := SellersPreviewBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}

	protoReq := BuildSellersPreviewRequest(body)
	if len(protoReq.SellerIds) != 2 || protoReq.SellerIds[0] != 7514 || protoReq.SellerIds[1] != 7515 {
		t.Fatalf("SellerIds = %v, want [7514 7515]", protoReq.SellerIds)
	}
}

func TestBindAndValidateSellersPreviewBodyAllowsEmptyIDs(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/sellers/preview", strings.NewReader(`{"seller_ids":[]}`))
	body := SellersPreviewBody{}

	if err := BindAndValidateJSON(req, &body); err != nil {
		t.Fatalf("BindAndValidateJSON returned error: %v", err)
	}
	if len(body.SellerIDs) != 0 {
		t.Fatalf("SellerIDs = %v, want empty", body.SellerIDs)
	}
}

func TestBindAndValidateSellersPreviewBodyRejectsNonPositiveIDs(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/sellers/preview", strings.NewReader(`{"seller_ids":[7514,0]}`))
	body := SellersPreviewBody{}

	err := BindAndValidateJSON(req, &body)
	if err == nil {
		t.Fatal("BindAndValidateJSON returned nil, want validation error")
	}

	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("error = %T, want validator.ValidationErrors", err)
	}
	if len(validationErrs) != 1 || !strings.HasPrefix(validationErrs[0].Field(), "seller_ids") || validationErrs[0].Tag() != "gt" {
		t.Fatalf("validation errors = %v, want seller_ids gt", validationErrs)
	}
}

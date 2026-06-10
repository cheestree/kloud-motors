package seller

import sellerpb "services/seller/proto"

type SellersPreviewBody struct {
	SellerIDs []int64 `json:"seller_ids" validate:"omitempty,dive,gt=0"`
}

func BuildGetSellerProfileRequest(sellerID int64) *sellerpb.GetSellerProfileRequest {
	return &sellerpb.GetSellerProfileRequest{SellerId: sellerID}
}

func BuildSellersPreviewRequest(body SellersPreviewBody) *sellerpb.SellersPreviewRequest {
	return &sellerpb.SellersPreviewRequest{SellerIds: body.SellerIDs}
}

package user

import userpb "services/user/proto"

type AuthBody struct {
	Email    string `json:"email" validate:"notblank"`
	Password string `json:"password" validate:"notblank"`
}

type RefreshTokenBody struct {
	RefreshToken string `json:"refresh_token" validate:"notblank"`
}

type UsersPreviewBody struct {
	UserIDs []int64 `json:"user_ids" validate:"omitempty,dive,gt=0"`
}

func BuildAuthRequest(body AuthBody) *userpb.AuthRequest {
	return &userpb.AuthRequest{
		Email:    body.Email,
		Password: body.Password,
	}
}

func BuildRefreshTokenRequest(body RefreshTokenBody) *userpb.RefreshTokenRequest {
	return &userpb.RefreshTokenRequest{RefreshToken: body.RefreshToken}
}

func BuildGetFavoritesRequest(userID int64) *userpb.GetFavoritesRequest {
	return &userpb.GetFavoritesRequest{UserId: userID}
}

func BuildAddFavoriteRequest(userID, listingID int64) *userpb.AddFavoriteRequest {
	return &userpb.AddFavoriteRequest{
		UserId:    userID,
		ListingId: listingID,
	}
}

func BuildRemoveFavoriteRequest(userID, listingID int64) *userpb.RemoveFavoriteRequest {
	return &userpb.RemoveFavoriteRequest{
		UserId:    userID,
		ListingId: listingID,
	}
}

func BuildUsersPreviewRequest(body UsersPreviewBody) *userpb.UsersPreviewRequest {
	return &userpb.UsersPreviewRequest{UserIds: body.UserIDs}
}

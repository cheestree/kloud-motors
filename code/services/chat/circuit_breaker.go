package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	listingproto "services/listing/proto"
	sellerproto "services/seller/proto"
	"services/shared"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newServiceCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		IsSuccessful: func(err error) bool {
			return !isBreakerFailure(err)
		},
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
}

const defaultGRPCCallTimeout = 5 * time.Second

func isBreakerFailure(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Internal, codes.Unknown, codes.DataLoss:
		return true
	default:
		return false
	}
}

func withBreaker[T any](ctx context.Context, breaker *gobreaker.CircuitBreaker, serviceName string, fn func(context.Context) (T, error)) (T, error) {
	var zero T

	callCtx, cancel := context.WithTimeout(ctx, defaultGRPCCallTimeout)
	defer cancel()

	result, err := breaker.Execute(func() (any, error) {
		return fn(callCtx)
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return zero, status.Error(codes.Unavailable, serviceName+" is unavailable")
		}
		return zero, err
	}

	typed, ok := result.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected breaker result type %T", result)
	}

	return typed, nil
}

type breakerListingClient struct {
	listingproto.ListingServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerListingClient(inner listingproto.ListingServiceClient) listingproto.ListingServiceClient {
	return &breakerListingClient{
		ListingServiceClient: inner,
		breaker:              newServiceCircuitBreaker("listing-service"),
	}
}

func (c *breakerListingClient) CheckListingOwnership(ctx context.Context, in *listingproto.CheckListingOwnershipRequest, opts ...grpc.CallOption) (*listingproto.CheckListingOwnershipResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingproto.CheckListingOwnershipResponse, error) {
		return c.ListingServiceClient.CheckListingOwnership(ctx, in, opts...)
	})
}

func (c *breakerListingClient) GetListingSummary(ctx context.Context, in *listingproto.ListingDetailsRequest, opts ...grpc.CallOption) (*shared.ListingSummary, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*shared.ListingSummary, error) {
		return c.ListingServiceClient.GetListingSummary(ctx, in, opts...)
	})
}

func (c *breakerListingClient) CheckListingOpen(ctx context.Context, in *listingproto.CheckListingOpenRequest, opts ...grpc.CallOption) (*listingproto.CheckListingOpenResponse, error) {
	return withBreaker(ctx, c.breaker, "listing service", func(ctx context.Context) (*listingproto.CheckListingOpenResponse, error) {
		return c.ListingServiceClient.CheckListingOpen(ctx, in, opts...)
	})
}

type breakerSellerClient struct {
	sellerproto.SellerServiceClient
	breaker *gobreaker.CircuitBreaker
}

func newBreakerSellerClient(inner sellerproto.SellerServiceClient) sellerproto.SellerServiceClient {
	return &breakerSellerClient{
		SellerServiceClient: inner,
		breaker:             newServiceCircuitBreaker("seller-service"),
	}
}

func (c *breakerSellerClient) VerifySellerProfile(ctx context.Context, in *sellerproto.VerifySellerRequest, opts ...grpc.CallOption) (*sellerproto.VerifySellerResponse, error) {
	return withBreaker(ctx, c.breaker, "seller service", func(ctx context.Context) (*sellerproto.VerifySellerResponse, error) {
		return c.SellerServiceClient.VerifySellerProfile(ctx, in, opts...)
	})
}

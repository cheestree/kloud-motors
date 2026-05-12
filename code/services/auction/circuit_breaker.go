package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	listingproto "services/listing/proto"

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
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})
}

func withBreaker[T any](breaker *gobreaker.CircuitBreaker, serviceName string, fn func() (T, error)) (T, error) {
	var zero T

	result, err := breaker.Execute(func() (any, error) {
		return fn()
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
	return withBreaker(c.breaker, "listing service", func() (*listingproto.CheckListingOwnershipResponse, error) {
		return c.ListingServiceClient.CheckListingOwnership(ctx, in, opts...)
	})
}

func (c *breakerListingClient) CheckListingOpen(ctx context.Context, in *listingproto.CheckListingOpenRequest, opts ...grpc.CallOption) (*listingproto.CheckListingOpenResponse, error) {
	return withBreaker(c.breaker, "listing service", func() (*listingproto.CheckListingOpenResponse, error) {
		return c.ListingServiceClient.CheckListingOpen(ctx, in, opts...)
	})
}
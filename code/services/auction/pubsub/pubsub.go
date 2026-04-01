package pubsub

import "context"

type PubSub interface {
	Publish(ctx context.Context, auctionID string, payload []byte) error
	Subscribe(auctionID string, handler func(payload []byte))
	Unsubscribe(auctionID string)
	Close() error
}

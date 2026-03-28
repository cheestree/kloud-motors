package pubsub

import "context"

type PubSub interface {
	Publish(ctx context.Context, chatID string, payload []byte) error
	Subscribe(chatID string, handler func(payload []byte))
	Unsubscribe(chatID string)
	Close() error
}

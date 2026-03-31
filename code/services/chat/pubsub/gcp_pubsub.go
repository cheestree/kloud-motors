package pubsub

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub"
)

type GCPPubSubConfig struct {
	ProjectID       string
	TopicID         string
	SubscriptionID  string
	NodeID          string
	CreateResources bool
}

type GCPPubSub struct {
	client *pubsub.Client
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	nodeID string

	mu       sync.RWMutex
	handlers map[string]func([]byte)

	runCtx    context.Context
	runCancel context.CancelFunc
	runWG     sync.WaitGroup
}

func NewGCPPubSub(ctx context.Context, cfg GCPPubSubConfig) (*GCPPubSub, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("missing GCP project id")
	}
	if cfg.TopicID == "" {
		return nil, fmt.Errorf("missing GCP topic id")
	}
	if cfg.SubscriptionID == "" {
		return nil, fmt.Errorf("missing GCP subscription id")
	}
	if cfg.NodeID == "" {
		return nil, fmt.Errorf("missing node id")
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}

	topic, err := ensureTopic(ctx, client, cfg.TopicID, cfg.CreateResources)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	topic.EnableMessageOrdering = true

	sub, err := ensureSubscription(ctx, client, topic, cfg.SubscriptionID, cfg.CreateResources)
	if err != nil {
		topic.Stop()
		_ = client.Close()
		return nil, err
	}

	runCtx, cancel := context.WithCancel(context.Background())

	ps := &GCPPubSub{
		client:    client,
		topic:     topic,
		sub:       sub,
		nodeID:    cfg.NodeID,
		handlers:  make(map[string]func([]byte)),
		runCtx:    runCtx,
		runCancel: cancel,
	}

	ps.runWG.Add(1)
	go ps.runReceiver()

	return ps, nil
}

func ensureTopic(ctx context.Context, client *pubsub.Client, topicID string, create bool) (*pubsub.Topic, error) {
	topic := client.Topic(topicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check topic exists: %w", err)
	}
	if exists {
		return topic, nil
	}
	if !create {
		return nil, fmt.Errorf("topic %q does not exist", topicID)
	}

	topic, err = client.CreateTopic(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("create topic: %w", err)
	}
	return topic, nil
}

func ensureSubscription(ctx context.Context, client *pubsub.Client, topic *pubsub.Topic, subID string, create bool) (*pubsub.Subscription, error) {
	sub := client.Subscription(subID)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check subscription exists: %w", err)
	}
	if exists {
		return sub, nil
	}
	if !create {
		return nil, fmt.Errorf("subscription %q does not exist", subID)
	}

	sub, err = client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{Topic: topic})
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return sub, nil
}

func (g *GCPPubSub) runReceiver() {
	defer g.runWG.Done()

	err := g.sub.Receive(g.runCtx, func(_ context.Context, msg *pubsub.Message) {
		chatID := msg.Attributes["chat_id"]
		origin := msg.Attributes["origin"]

		if chatID == "" {
			msg.Ack()
			return
		}

		if origin == g.nodeID {
			msg.Ack()
			return
		}

		g.mu.RLock()
		handler := g.handlers[chatID]
		g.mu.RUnlock()
		if handler == nil {
			msg.Ack()
			return
		}

		handler(msg.Data)
		msg.Ack()
	})
	if err != nil {
		fmt.Printf("pubsub receive stopped: %v\n", err)
	}
}

func (g *GCPPubSub) Publish(ctx context.Context, chatID string, payload []byte) error {
	result := g.topic.Publish(ctx, &pubsub.Message{
		Data:        payload,
		OrderingKey: chatID,
		Attributes: map[string]string{
			"chat_id": chatID,
			"origin":  g.nodeID,
		},
	})

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish gcp pubsub event: %w", err)
	}
	return nil
}

func (g *GCPPubSub) Subscribe(chatID string, handler func(payload []byte)) {
	g.mu.Lock()
	g.handlers[chatID] = handler
	g.mu.Unlock()
}

func (g *GCPPubSub) Unsubscribe(chatID string) {
	g.mu.Lock()
	delete(g.handlers, chatID)
	g.mu.Unlock()
}

func (g *GCPPubSub) Close() error {
	g.runCancel()
	g.runWG.Wait()
	g.topic.Stop()
	return g.client.Close()
}

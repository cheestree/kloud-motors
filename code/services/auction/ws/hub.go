package ws

import (
	"context"
	"services/auction/pubsub"
	"sync"
)

type Hub struct {
	rooms  map[string]*Room
	mu     sync.RWMutex
	pubsub pubsub.PubSub
}

func NewHub(pubsub pubsub.PubSub) *Hub {
	return &Hub{
		rooms:  make(map[string]*Room),
		pubsub: pubsub,
	}
}

func (h *Hub) getOrCreateRoom(auctionID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	r, ok := h.rooms[auctionID]
	if !ok {
		r = newRoom(func() {
			h.onRoomEmpty(auctionID)
		})
		go r.run()
		h.rooms[auctionID] = r

		if h.pubsub != nil {
			h.pubsub.Subscribe(auctionID, func(msg []byte) {
				h.BroadcastLocal(auctionID, msg)
			})
		}
	}

	return r
}

func (h *Hub) Register(auctionID string, c *Client) {
	h.getOrCreateRoom(auctionID).register <- c
}

func (h *Hub) Unregister(auctionID string, c *Client) {
	h.mu.RLock()
	r, ok := h.rooms[auctionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.unregister <- c
}

func (h *Hub) BroadcastLocal(auctionID string, msg []byte) {
	h.mu.RLock()
	r, ok := h.rooms[auctionID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.broadcast <- msg
}

func (h *Hub) Publish(auctionID string, msg []byte) error {
	h.BroadcastLocal(auctionID, msg)

	if h.pubsub == nil {
		return nil
	}

	return h.pubsub.Publish(context.Background(), auctionID, msg)
}

func (h *Hub) onRoomEmpty(auctionID string) {
	h.mu.Lock()
	delete(h.rooms, auctionID)
	h.mu.Unlock()

	if h.pubsub != nil {
		h.pubsub.Unsubscribe(auctionID)
	}
}
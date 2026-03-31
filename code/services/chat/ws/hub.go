package ws

import (
	pubsub2 "services/chat/pubsub"
	"context"
	"sync"
)

type Hub struct {
	rooms  map[string]*Room
	mu     sync.RWMutex
	pubsub pubsub2.PubSub
}

func NewHub(pubsub pubsub2.PubSub) *Hub {
	return &Hub{
		rooms:  make(map[string]*Room),
		pubsub: pubsub,
	}
}

func (h *Hub) getOrCreateRoom(chatID string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	r, ok := h.rooms[chatID]
	if !ok {
		r = newRoom(func() {
			h.onRoomEmpty(chatID)
		})
		go r.run()
		h.rooms[chatID] = r

		if h.pubsub != nil {
			h.pubsub.Subscribe(chatID, func(msg []byte) {
				h.BroadcastLocal(chatID, msg)
			})
		}
	}

	return r
}

func (h *Hub) Register(chatID string, c *Client) {
	h.getOrCreateRoom(chatID).register <- c
}

func (h *Hub) Unregister(chatID string, c *Client) {
	h.mu.RLock()
	r, ok := h.rooms[chatID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.unregister <- c
}

func (h *Hub) BroadcastLocal(chatID string, msg []byte) {
	h.mu.RLock()
	r, ok := h.rooms[chatID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.broadcast <- msg
}

func (h *Hub) Publish(chatID string, msg []byte) error {
	h.BroadcastLocal(chatID, msg)

	if h.pubsub == nil {
		return nil
	}

	return h.pubsub.Publish(context.Background(), chatID, msg)
}

func (h *Hub) onRoomEmpty(chatID string) {
	h.mu.Lock()
	if _, ok := h.rooms[chatID]; ok {
		delete(h.rooms, chatID)
	}
	h.mu.Unlock()

	if h.pubsub != nil {
		h.pubsub.Unsubscribe(chatID)
	}
}

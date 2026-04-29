package ws

const maxRoomClients = 100

type Room struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	onEmpty    func()
}

func newRoom(onEmpty func()) *Room {
	return &Room{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		onEmpty:    onEmpty,
	}
}

func (r *Room) run() {
	for {
		select {
		case c := <-r.register:
			if len(r.clients) >= maxRoomClients {
				close(c.send)
				_ = c.conn.Close()
				continue
			}
			r.clients[c] = struct{}{}

		case c := <-r.unregister:
			if _, ok := r.clients[c]; ok {
				delete(r.clients, c)
				close(c.send)
			}

			if len(r.clients) == 0 {
				if r.onEmpty != nil {
					r.onEmpty()
				}
				return
			}

		case msg := <-r.broadcast:
			for c := range r.clients {
				select {
				case c.send <- msg:
				default:
					close(c.send)
					delete(r.clients, c)
				}
			}

			if len(r.clients) == 0 {
				if r.onEmpty != nil {
					r.onEmpty()
				}
				return
			}
		}
	}
}

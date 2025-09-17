package serverside

import (
	"fmt"
	"sync"
)

type ChatRoom struct {
	Id       string
	Name     string
	Clients  map[*Client]bool
	Messages chan Message
	Mu       sync.Mutex
}

func (r *ChatRoom) Broadcaster() {
	for msg := range r.Messages {
		r.Mu.Lock()
		for c := range r.Clients {
			if c != msg.Sender {
				select {
				case c.Ch <- msg.Text:
				default:
				}
			}
		}
		r.Mu.Unlock()
	}
}

func (r *ChatRoom) AddClient(client *Client) {
	r.Mu.Lock()
	r.Clients[client] = true
	client.Room = r
	r.Mu.Unlock()

	// Notify room about new user
	r.Messages <- Message{
		Sender: client,
		Text:   fmt.Sprintf("*** User%s joined %s ***", client.Username, r.Name),
	}
}

func (r *ChatRoom) RemoveClient(client *Client) {
	r.Mu.Lock()
	delete(r.Clients, client)
	r.Mu.Unlock()

	// Notify room about user leaving
	r.Messages <- Message{
		Sender: client,
		Text:   fmt.Sprintf("*** User%s left %s ***", client.Username, r.Name),
	}
}

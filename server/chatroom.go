package main

import (
	"fmt"
	"sync"
)

type ChatRoom struct {
	id       string
	name     string
	clients  map[*Client]bool
	messages chan Message
	mu       sync.Mutex
}

func (r *ChatRoom) Broadcaster() {
	for msg := range r.messages {
		r.mu.Lock()
		for c := range r.clients {
			if c != msg.sender {
				select {
				case c.ch <- msg.text:
				default:
				}
			}
		}
		r.mu.Unlock()
	}
}

func (r *ChatRoom) AddClient(client *Client) {
	r.mu.Lock()
	r.clients[client] = true
	client.room = r
	r.mu.Unlock()

	// Notify room about new user
	r.messages <- Message{
		sender: client,
		text:   fmt.Sprintf("*** User%s joined %s ***", client.username, r.name),
	}
}

func (r *ChatRoom) RemoveClient(client *Client) {
	r.mu.Lock()
	delete(r.clients, client)
	r.mu.Unlock()

	// Notify room about user leaving
	r.messages <- Message{
		sender: client,
		text:   fmt.Sprintf("*** User%s left %s ***", client.username, r.name),
	}
}

package main

import (
	"fmt"
	"sync"
)

type Server struct {
	rooms     map[string]*ChatRoom
	mu        sync.RWMutex
	idCounter int64
}

func (s *Server) CreateRoom(id, name string) {
	s.mu.Lock()
	room := &ChatRoom{name: name, id: id, clients: make(map[*Client]bool), messages: make(chan Message)}

	s.rooms[id] = room
	go room.Broadcaster()

	s.mu.Unlock()
}

func (s *Server) GetRooms() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rooms []string
	for id, room := range s.rooms {
		clientCount := len(room.clients)
		rooms = append(rooms, fmt.Sprintf("%s:%s (%d users)", id, room.name, clientCount))
	}
	return rooms
}

func (s *Server) validateUsername(username string, currentClient *Client) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, room := range s.rooms {
		room.mu.Lock()
		for client := range room.clients {
			if client != currentClient && client.username == username {
				room.mu.Unlock()
				return false
			}
		}
		room.mu.Unlock()

	}
	return true
}

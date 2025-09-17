package serverside

import (
	"fmt"
	"sync"
)

type Server struct {
	Rooms     map[string]*ChatRoom
	Mu        sync.RWMutex
	IdCounter int64
}

func (s *Server) CreateRoom(id, name string) {
	s.Mu.Lock()
	room := &ChatRoom{Name: name, Id: id, Clients: make(map[*Client]bool), Messages: make(chan Message)}

	s.Rooms[id] = room
	go room.Broadcaster()

	s.Mu.Unlock()
}

func (s *Server) GetRooms() []string {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	var rooms []string
	for id, room := range s.Rooms {
		clientCount := len(room.Clients)
		rooms = append(rooms, fmt.Sprintf("%s:%s (%d users)", id, room.Name, clientCount))
	}
	return rooms
}

func (s *Server) ValidateUsername(username string, currentClient *Client) bool {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	for _, room := range s.Rooms {
		room.Mu.Lock()
		for client := range room.Clients {
			if client != currentClient && client.Username == username {
				room.Mu.Unlock()
				return false
			}
		}
		room.Mu.Unlock()

	}
	return true
}

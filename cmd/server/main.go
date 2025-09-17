package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"
	_ "github.com/mattn/go-sqlite3"

	"github.com/MukalDadhwal/BubbleChat/serverside"
)



var server *serverside.Server

func init() {
	server = &serverside.Server{
		Rooms: make(map[string]*serverside.ChatRoom),
	}
	server.CreateRoom("general", "General Chat")

}

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// assign unique ID
	clientId := atomic.AddInt64(&server.IdCounter, 1)
	client := &serverside.Client{
		Id:   clientId,
		Conn: conn,
		Ch:   make(chan string, 10),
	}
	log.Printf("New connection: User%d from %s", client.Id, conn.RemoteAddr().String())

	go func() {
		writer := bufio.NewWriter(conn)
		for msg := range client.Ch {
			fmt.Fprintln(writer, msg)
			writer.Flush()
		}
	}()

	client.Ch <- "=== Welcome to BubbleChat ==="
	client.Ch <- "Commands:"
	client.Ch <- " /list - Show available rooms"
	client.Ch <- " /join <room_id> - Join a room"
	client.Ch <- " /leave - Leave current room"
	client.Ch <- " /create <room_name> - Create new room"

	showRoomList(client)

	// Reading messages from conn
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		// Handle commands
		parts := strings.SplitN(text, " ", 2)
		command := parts[0]
		// fmt.Println(command, len(command))

		switch command {
		case "/list":
			showRoomList(client)
		case "/join":
			if len(parts) > 1 {
				joinRoom(client, parts[1])
			} else {
				client.Ch <- "Usage: /join <room_id>"
			}
		case "/username":
			if len(parts) > 1 {
				changeUsername(client, parts[1])
			} else {
				client.Ch <- "Usage: /username <new_name>"
			}
		case "/leave":
			leaveRoom(client)
		case "/create":
			if len(parts) > 1 {
				createRoom(client, parts[1])
			} else {
				client.Ch <- "Usage: /create <room_id>"
			}

		default:
			// Regular chat message (only if in a room)
			if client.Room != nil {
				client.Room.Messages <- serverside.Message{
					Sender: client,
					Text:   fmt.Sprintf("User%s: %s", client.Username, text),
				}
			} else {
				client.Ch <- "Please join a room first. Use /list to see available rooms."
			}
		}

	}
	// Cleanup when client disconnects
	if client.Room != nil {
		client.Room.RemoveClient(client)
	}
	close(client.Ch) // Close channel only on disconnect
}

func showRoomList(client *serverside.Client) {
	rooms := server.GetRooms()
	client.Ch <- "=== Available Rooms ==="
	if len(rooms) == 0 {
		client.Ch <- "No rooms available"
	} else {
		for _, room := range rooms {
			client.Ch <- " " + room
		}
	}
	client.Ch <- "========================"
}

func joinRoom(client *serverside.Client, roomId string) {
	server.Mu.RLock()
	room, exists := server.Rooms[roomId]
	server.Mu.RUnlock()

	if !exists {
		client.Ch <- fmt.Sprintf("Room '%s' does not exits. Use /list to see availble rooms", roomId)
		return
	}

	if client.Room != nil {
		client.Room.RemoveClient(client)
	}

	room.AddClient(client)
	client.Ch <- fmt.Sprintf("*** Joined room: %s ***", room.Name)
}

func createRoom(client *serverside.Client, roomName string) {
	roomId := strings.ToLower(strings.ReplaceAll(roomName, " ", "_"))

	server.Mu.RLock()
	_, exists := server.Rooms[roomId]
	server.Mu.RUnlock()

	if exists {
		client.Ch <- fmt.Sprintf("Room '%s' already exists", roomId)
		return
	}

	server.CreateRoom(roomId, roomName)
	client.Ch <- fmt.Sprintf("*** Created room: %s ***", roomName)

	// Auto-join the created room
	joinRoom(client, roomId)
}

func leaveRoom(client *serverside.Client) {
	if client.Room == nil {
		client.Ch <- "You are not in any room"
		return
	}

	roomName := client.Room.Name
	client.Room.RemoveClient(client)
	client.Room = nil
	client.Ch <- fmt.Sprintf("*** Left room: %s ***", roomName)
}

// Complete your existing changeUsername function:
func changeUsername(client *serverside.Client, newUsername string) {
	newUsername = strings.TrimSpace(newUsername)

	if newUsername == "" {
		client.Ch <- "Username cannot be empty"
		return
	}

	if newUsername == client.Username {
		client.Ch <- "That's already your username"
		return
	}

	if !server.ValidateUsername(newUsername, client) {
		client.Ch <- fmt.Sprintf("Username '%s' is already taken", newUsername)
		return
	}

	oldUsername := client.Username
	client.Username = newUsername

	client.Ch <- fmt.Sprintf("*** Username changed from '%s' to '%s' ***", oldUsername, newUsername)

	if client.Room != nil {
		client.Room.Messages <- serverside.Message{
			Sender: client,
			Text:   fmt.Sprintf("*** %s is now known as %s ***", oldUsername, newUsername),
		}
	}

	log.Printf("User %d changed username from '%s' to '%s'", client.Id, oldUsername, newUsername)
}

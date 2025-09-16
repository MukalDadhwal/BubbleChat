package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"
    _ "github.com/mattn/go-sqlite3"
)

type Client struct {
	id       int64
	username string
	conn     net.Conn
	ch       chan string
	room     *ChatRoom
}

type Message struct {
	sender *Client
	text   string
}

var server *Server

func init() {
	server = &Server{
		rooms: make(map[string]*ChatRoom),
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
	clientId := atomic.AddInt64(&server.idCounter, 1)
	client := &Client{
		id:   clientId,
		conn: conn,
		ch:   make(chan string, 10),
	}
	log.Printf("New connection: User%d from %s", client.id, conn.RemoteAddr().String())

	go func() {
		writer := bufio.NewWriter(conn)
		for msg := range client.ch {
			fmt.Fprintln(writer, msg)
			writer.Flush()
		}
	}()

	client.ch <- "=== Welcome to BubbleChat ==="
	client.ch <- "Commands:"
	client.ch <- " /list - Show available rooms"
	client.ch <- " /join <room_id> - Join a room"
	client.ch <- " /leave - Leave current room"
	client.ch <- " /create <room_name> - Create new room"

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
				client.ch <- "Usage: /join <room_id>"
			}
		case "/username":
			if len(parts) > 1 {
				changeUsername(client, parts[1])
			} else {
				client.ch <- "Usage: /username <new_name>"
			}
		case "/leave":
			leaveRoom(client)
		case "/create":
			if len(parts) > 1 {
				createRoom(client, parts[1])
			} else {
				client.ch <- "Usage: /create <room_id>"
			}

		default:
			// Regular chat message (only if in a room)
			if client.room != nil {
				client.room.messages <- Message{
					sender: client,
					text:   fmt.Sprintf("User%s: %s", client.username, text),
				}
			} else {
				client.ch <- "Please join a room first. Use /list to see available rooms."
			}
		}

	}
	// Cleanup when client disconnects
	if client.room != nil {
		client.room.RemoveClient(client)
	}
	close(client.ch) // Close channel only on disconnect
}

func showRoomList(client *Client) {
	rooms := server.GetRooms()
	client.ch <- "=== Available Rooms ==="
	if len(rooms) == 0 {
		client.ch <- "No rooms available"
	} else {
		for _, room := range rooms {
			client.ch <- " " + room
		}
	}
	client.ch <- "========================"
}

func joinRoom(client *Client, roomId string) {
	server.mu.RLock()
	room, exists := server.rooms[roomId]
	server.mu.RUnlock()

	if !exists {
		client.ch <- fmt.Sprintf("Room '%s' does not exits. Use /list to see availble rooms", roomId)
		return
	}

	if client.room != nil {
		client.room.RemoveClient(client)
	}

	room.AddClient(client)
	client.ch <- fmt.Sprintf("*** Joined room: %s ***", room.name)
}

func createRoom(client *Client, roomName string) {
	roomId := strings.ToLower(strings.ReplaceAll(roomName, " ", "_"))

	server.mu.RLock()
	_, exists := server.rooms[roomId]
	server.mu.RUnlock()

	if exists {
		client.ch <- fmt.Sprintf("Room '%s' already exists", roomId)
		return
	}

	server.CreateRoom(roomId, roomName)
	client.ch <- fmt.Sprintf("*** Created room: %s ***", roomName)

	// Auto-join the created room
	joinRoom(client, roomId)
}

func leaveRoom(client *Client) {
	if client.room == nil {
		client.ch <- "You are not in any room"
		return
	}

	roomName := client.room.name
	client.room.RemoveClient(client)
	client.room = nil
	client.ch <- fmt.Sprintf("*** Left room: %s ***", roomName)
}

// Complete your existing changeUsername function:
func changeUsername(client *Client, newUsername string) {
	newUsername = strings.TrimSpace(newUsername)

	if newUsername == "" {
		client.ch <- "Username cannot be empty"
		return
	}

	if newUsername == client.username {
		client.ch <- "That's already your username"
		return
	}

	if !server.validateUsername(newUsername, client) {
		client.ch <- fmt.Sprintf("Username '%s' is already taken", newUsername)
		return
	}

	oldUsername := client.username
	client.username = newUsername

	client.ch <- fmt.Sprintf("*** Username changed from '%s' to '%s' ***", oldUsername, newUsername)

	if client.room != nil {
		client.room.messages <- Message{
			sender: client,
			text:   fmt.Sprintf("*** %s is now known as %s ***", oldUsername, newUsername),
		}
	}

	log.Printf("User %d changed username from '%s' to '%s'", client.id, oldUsername, newUsername)
}

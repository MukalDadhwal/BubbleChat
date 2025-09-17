package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const CONFIG_FILE = "username.txt"

func main() {
	// Explicitly call getUsername to ensure it's used
	username, err := getUsername()
	if err != nil {
		log.Fatal("Failed to get username: ", err)
	}

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("Failed to connect to server:", err)
	}
	defer conn.Close()

	fmt.Println("Connected to BubbleChat server!")
	fmt.Println("Type /help to see commands")

	writer := bufio.NewWriter(conn)
	fmt.Fprintln(writer, username)
	writer.Flush()

	currentRoom := ""

	// read from server
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			message := scanner.Text()
			fmt.Println(message)

			// Update current room based on server messages
			if strings.Contains(message, "*** Joined room:") {
				parts := strings.Split(message, ":")
				if len(parts) > 1 {
					currentRoom = strings.TrimSpace(strings.Replace(parts[1], "***", "", -1))
				}
			} else if strings.Contains(message, "*** Left room:") {
				currentRoom = ""
			}
		}
		if err := scanner.Err(); err != nil {
			log.Println("server read error:", err)
			os.Exit(0) // server closed
		}
	}()

	// Read user input and send to server
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Show current room in prompt
		if currentRoom != "" {
			fmt.Printf("[%s]> ", currentRoom)
		} else {
			fmt.Print("[lobby]> ")
		}

		if !scanner.Scan() {
			break
		}

		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		if text == "/help" {
			showHelp()
			continue
		}

		if text == "/room" || text == "/current" {
			if currentRoom != "" {
				fmt.Printf("Currently in room: %s\n", currentRoom)
			} else {
				fmt.Println("Not in any room (lobby)")
			}
			continue
		}

		if text == "/quit" || text == "/exit" {
			fmt.Println("Goodbye!")
			break
		}

		// Ensure saveUsername is called when updating the username
		if strings.HasPrefix(text, "/username ") {
			parts := strings.SplitN(text, " ", 2)
			if len(parts) > 1 {
				newUsername := strings.TrimSpace(parts[1])
				if newUsername != "" {
					saveUsername(newUsername)
					fmt.Println("Username saved: ", newUsername) // Debugging line to ensure usage
				}
			}
		}

		fmt.Fprintln(writer, text) // Change from fmt.Println to fmt.Fprintln
		writer.Flush()
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading input:", err)
	}
}

func getUsername() (string, error) {
	// Try to read from config file
	if data, err := os.ReadFile(CONFIG_FILE); err == nil {
		savedUsername := strings.TrimSpace(string(data))
		if savedUsername != "" {
			fmt.Printf("Welcome back! Your username is: %s\n", savedUsername)
			fmt.Print("Press Enter to use it, or type a new username: ")

			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				input := strings.TrimSpace(scanner.Text())
				if input == "" {
					return savedUsername, nil
				} else {
					saveUsername(input)
					return input, nil
				}
			}
		}
	}

	// No saved username, prompt for new one
	fmt.Print("Enter your username: ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		username := strings.TrimSpace(scanner.Text())
		if username != "" {
			saveUsername(username)
			return username, nil
		}
		fmt.Print("Username cannot be empty. Please try again: ")
	}

	return "", fmt.Errorf("failed to get username")
}

func saveUsername(username string) error {
	return os.WriteFile(CONFIG_FILE, []byte(username), 0644)
}

func showHelp() {
	fmt.Println("=== BubbleChat Client Help ===")
	fmt.Println("Server Commands:")
	fmt.Println("  /list                - Show available rooms")
	fmt.Println("  /join <room_id>      - Join a room")
	fmt.Println("  /leave               - Leave current room")
	fmt.Println("  /create <room_name>  - Create new room")
	fmt.Println("  /username <new_name> - Change username")
	fmt.Println()
	fmt.Println("Client Commands:")
	fmt.Println("  /help                - Show this help")
	fmt.Println("  /room or /current    - Show current room")
	fmt.Println("  /quit or /exit       - Disconnect")
	fmt.Println()
	fmt.Println("Once in a room, just type messages to chat!")
	fmt.Println("===============================")
}

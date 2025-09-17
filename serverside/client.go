package serverside

import "net"

type Client struct {
	Id       int64
	Username string
	Conn     net.Conn
	Ch       chan string
	Room     *ChatRoom
}

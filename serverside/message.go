package serverside

type Message struct {
	Sender *Client
	Text   string
}
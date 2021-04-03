package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

type ClientManager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}
type Client struct {
	id     string
	socket *websocket.Conn
	sent   chan []byte
}
type Message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

var manager = ClientManager{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

func (manager *ClientManager) start() {
	for {
		select {
		case conn := <-manager.register:
			manager.clients[conn] = true
			jsonMessage, err := json.Marshal(
				&Message{
					Content: "/A new socket has connected.",
				},
			)
			if err != nil {
				log.Println(err)
				return
			}
			manager.send(jsonMessage, conn)
		case conn := <-manager.unregister:
			if _, ok := manager.clients[conn]; ok {
				close(conn.sent)
				delete(manager.clients, conn)
				jsonMessage, err := json.Marshal(
					&Message{
						Content: "/A socket has disconnected.",
					},
				)
				if err != nil {
					log.Println(err)
					return
				}
				manager.send(jsonMessage, conn)
			}
		case message := <-manager.broadcast:
			for conn := range manager.clients {
				select {
				case conn.sent <- message:
				default:
					close(conn.sent)
					delete(manager.clients, conn)
				}
			}
		}
	}
}
func (manager *ClientManager) send(message []byte, ignore *Client) {
	for conn := range manager.clients {
		if conn != ignore {
			conn.sent <- message
		}
	}
}
func (c *Client) read() {
	defer func() {
		manager.unregister <- c
		if err := c.socket.Close(); err != nil {
			log.Println(err)
			return
		}
	}()
	for {
		_, message, err := c.socket.ReadMessage()
		if err != nil {
			manager.unregister <- c
			if serr := c.socket.Close(); serr != nil {
				log.Println(serr)
				return
			}
			break
		}
		jsonMessage, err := json.Marshal(
			&Message{
				Sender:  c.id,
				Content: string(message),
			},
		)
		manager.broadcast <- jsonMessage
	}
}
func (c *Client) write() {
	defer func() {
		if err := c.socket.Close(); err != nil {
			log.Println(err)
			return
		}
	}()
	for {
		select {
		case message, ok := <-c.sent:
			if !ok {
				if err := c.socket.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					log.Println(err)
					return
				}
			}
			if err := c.socket.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println(err)
				return
			}
		}
	}
}
func main() {
	fmt.Println("Starting application...")
	go manager.start()
	http.HandleFunc("/ws", wsPage)
	if err := http.ListenAndServe(":12345", nil); err != nil {
		log.Fatal(err)
	}
}
func wsPage(w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	newV4, err := uuid.NewV4()
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{
		id:     newV4.String(),
		socket: conn,
		sent:   make(chan []byte),
	}
	manager.register <- client
	go client.read()
	go client.write()
}

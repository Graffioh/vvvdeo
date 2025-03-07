package websocket

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Key string `json:"videoKey"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var KeySocketConnections = make(map[string]*websocket.Conn)

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	var msg WebSocketMessage

	err = conn.ReadJSON(&msg)
	if err != nil {
		log.Println("Failed to read WebSocket message:", err)
		return
	}

	videoKey := strings.Split(msg.Key, "/")[1]

	KeySocketConnections[videoKey] = conn
}

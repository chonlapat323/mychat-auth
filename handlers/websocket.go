package handlers

import (
	"encoding/json"
	"log"
	"mychat-auth/utils"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// roomID -> list of connections
var roomConnections = make(map[string]map[*websocket.Conn]string)
var mu sync.Mutex

// MessageEvent represents incoming WebSocket messages
type MessageEvent struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
	Text   string `json:"text,omitempty"`
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	token = strings.TrimPrefix(token, "Bearer ")
	claims, err := utils.ValidateToken(token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	userID := claims.UserID
	log.Printf("User %s connected via WebSocket", userID)

	var currentRoom string

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		var event MessageEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		switch event.Type {
		case "join":
			mu.Lock()
			if _, ok := roomConnections[event.RoomID]; !ok {
				rm := make(map[*websocket.Conn]string)
				rm[conn] = userID
				roomConnections[event.RoomID] = rm
			} else {
				roomConnections[event.RoomID][conn] = userID
			}
			mu.Unlock()
			currentRoom = event.RoomID
			log.Printf("User %s joined room %s", userID, currentRoom)

		case "message":
			if event.RoomID == "" || event.Text == "" {
				continue
			}
			broadcastToRoom(event.RoomID, userID, event.Text)
		}
	}

	// Cleanup on disconnect
	if currentRoom != "" {
		mu.Lock()
		delete(roomConnections[currentRoom], conn)
		mu.Unlock()
	}
}

func broadcastToRoom(roomID, senderID, text string) {
	mu.Lock()
	conns := roomConnections[roomID]
	mu.Unlock()

	message := map[string]string{
		"user_id": senderID,
		"text":    text,
	}
	data, _ := json.Marshal(message)

	for conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("Write error:", err)
			conn.Close()
			mu.Lock()
			delete(roomConnections[roomID], conn)
			mu.Unlock()
		}
	}
}

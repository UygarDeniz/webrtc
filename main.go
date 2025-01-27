package main

import (
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // Allow all origins for development
	}
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("Attempting to upgrade connection")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	log.Println("WebSocket connection established")
	defer func() {
		conn.Close()
		clientsMu.Lock()
		delete(clients, conn)
		clientsMu.Unlock()
		log.Println("Client disconnected")
	}()

	// Add client to the clients map
	clientsMu.Lock()
	clients[conn] = true
	clientsMu.Unlock()
	log.Println("New client connected. Total clients:", len(clients))

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			} else {
				log.Printf("Read error: %v", err)
			}
			break
		}

		log.Printf("Received message: %s", msg)

		// Broadcast the message to all connected clients except the sender
		clientsMu.Lock()
		for client := range clients {
			if client != conn {
				err := client.WriteMessage(messageType, msg)
				if err != nil {
					log.Println("Write error:", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		clientsMu.Unlock()
	}

	log.Println("Exiting handleWebSocket loop")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/ws", handleWebSocket)
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

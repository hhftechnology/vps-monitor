// home/main.go
package main

import (

	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Metrics holds the structure for incoming data from the agent.
type Metrics struct {
	Hostname  string `json:"hostname"`
	// Add all other fields from the agent's Metric struct here
}

var (
	// upgrader is used to upgrade HTTP connections to WebSocket connections.
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow all connections for simplicity. In production, you'd want to
			// restrict this to your frontend's domain.
			return true
		},
	}
	// latestMetrics stores the most recent data received from the agent.
	// We use a mutex to prevent race conditions when accessing it.
	latestMetrics *Metrics
	mu            sync.RWMutex
	// clients is a map of all active WebSocket clients.
	clients = make(map[*websocket.Conn]bool)
	// broadcast is a channel to send metrics to all connected clients.
	broadcast = make(chan *Metrics)
)

func main() {
	router := gin.Default()

	// API endpoint for the agent to post data to.
	router.POST("/api/metrics", handleMetricsPost)

	// WebSocket endpoint for the frontend to connect to.
	router.GET("/ws", handleWebSocket)

	// Serve the static frontend files.
	router.Static("/", "./web/build")

	// Start the broadcast loop in a separate goroutine.
	go handleBroadcast()

	log.Println("Home server starting on :8085")
	router.Run(":8085")
}

// handleMetricsPost handles incoming data from the agent.
func handleMetricsPost(c *gin.Context) {
	var metrics Metrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store the latest metrics.
	mu.Lock()
	latestMetrics = &metrics
	mu.Unlock()

	// Send the new metrics to the broadcast channel.
	broadcast <- &metrics

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleWebSocket handles new WebSocket connections from the frontend.
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	// Register new client.
	clients[conn] = true
	log.Println("New client connected.")

	// Send the last known metrics to the new client immediately.
	mu.RLock()
	if latestMetrics != nil {
		if err := conn.WriteJSON(latestMetrics); err != nil {
			log.Printf("Error sending initial metrics: %v", err)
		}
	}
	mu.RUnlock()

	// Keep the connection open. The broadcast loop will handle sending new data.
	for {
		// We can read messages from the client here if needed, but for now, we'll just keep it open.
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Client disconnected: %v", err)
			delete(clients, conn)
			break
		}
	}
}

// handleBroadcast listens on the broadcast channel and sends data to all clients.
func handleBroadcast() {
	for {
		metrics := <-broadcast
		// Send to all connected clients.
		for client := range clients {
			err := client.WriteJSON(metrics)
			if err != nil {
				log.Printf("Error writing to client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

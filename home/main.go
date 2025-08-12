// home/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// ProcessInfo holds simplified information about a running process.
type ProcessInfo struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
}

// AgentInfo holds information about the agent itself
type AgentInfo struct {
	Version       string            `json:"version"`
	GoVersion     string            `json:"go_version"`
	NumGoroutines int               `json:"num_goroutines"`
	MemStats      map[string]uint64 `json:"mem_stats"`
}

// Metrics holds all the system metrics we want to collect.
type Metrics struct {
	AgentID       string                   `json:"agent_id"`
	Hostname      string                   `json:"hostname"`
	Uptime        uint64                   `json:"uptime"`
	CPUUsage      float64                  `json:"cpu_usage"`
	Memory        *mem.VirtualMemoryStat   `json:"memory"`
	Disk          *disk.UsageStat          `json:"disk"`
	Network       []net.IOCountersStat     `json:"network"`
	Processes     []*ProcessInfo           `json:"processes"`
	AgentInfo     *AgentInfo               `json:"agent_info"`
	Timestamp     time.Time                `json:"timestamp"`
	LastSeen      time.Time                `json:"last_seen"`
}

// AgentSummary provides overview information about an agent
type AgentSummary struct {
	AgentID     string    `json:"agent_id"`
	Hostname    string    `json:"hostname"`
	LastSeen    time.Time `json:"last_seen"`
	IsOnline    bool      `json:"is_online"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	Uptime      uint64    `json:"uptime"`
}

// MultiAgentData contains data for all agents
type MultiAgentData struct {
	Agents   map[string]*Metrics `json:"agents"`
	Summary  []*AgentSummary     `json:"summary"`
	Timestamp time.Time          `json:"timestamp"`
}

var (
	// upgrader is used to upgrade HTTP connections to WebSocket connections.
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all connections for simplicity. In production, you'd want to
			// restrict this to your frontend's domain.
			return true
		},
	}
	// agentMetrics stores metrics for each agent by agent ID
	agentMetrics = make(map[string]*Metrics)
	// mu protects agentMetrics map
	mu sync.RWMutex
	// clients is a map of all active WebSocket clients.
	clients = make(map[*websocket.Conn]bool)
	// broadcast is a channel to send metrics to all connected clients.
	broadcast = make(chan *MultiAgentData, 10) // Buffered channel
	// Agent timeout duration
	agentTimeout = 2 * time.Minute
)

func main() {
	// Set Gin to release mode
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	
	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Group all API routes under /api
	api := router.Group("/api")
	{
		// API endpoint for the agent to post data to.
		api.POST("/metrics", handleMetricsPost)

		// WebSocket endpoint for the frontend to connect to.
		api.GET("/ws", handleWebSocket)
		
		// Health check endpoint
		api.GET("/health", func(c *gin.Context) {
			mu.RLock()
			agentCount := len(agentMetrics)
			onlineAgents := 0
			for _, metrics := range agentMetrics {
				if time.Since(metrics.LastSeen) < agentTimeout {
					onlineAgents++
				}
			}
			mu.RUnlock()
			
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"agents": gin.H{
					"total":  agentCount,
					"online": onlineAgents,
				},
			})
		})
		
		// Get list of all agents
		api.GET("/agents", handleGetAgents)
		
		// Get specific agent data
		api.GET("/agents/:agentId", handleGetAgent)
	}

	// Serve static files with custom handler to avoid route conflicts
	router.Use(staticFileHandler("./web/build"))

	// Start the broadcast loop in a separate goroutine.
	go handleBroadcast()
	
	// Start agent cleanup routine
	go agentCleanupRoutine()

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:         ":8085",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Home server starting on :8085")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give a 30-second timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close all WebSocket connections
	for client := range clients {
		client.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// staticFileHandler serves static files only if the path doesn't start with /api
func staticFileHandler(root string) gin.HandlerFunc {
	fileServer := http.FileServer(http.Dir(root))
	
	return func(c *gin.Context) {
		// Skip if this is an API route
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}

		// Check if file exists
		path := filepath.Join(root, c.Request.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// If file doesn't exist, serve index.html (for SPA routing)
			path = filepath.Join(root, "index.html")
		}

		// Serve the file
		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/")
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// handleMetricsPost handles incoming data from the agent.
func handleMetricsPost(c *gin.Context) {
	var metrics Metrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		log.Printf("Invalid JSON from agent: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Add timestamps
	metrics.Timestamp = time.Now()
	metrics.LastSeen = time.Now()
	
	// Use AgentID if provided, otherwise use hostname
	if metrics.AgentID == "" {
		metrics.AgentID = metrics.Hostname
	}

	// Store the metrics for this agent
	mu.Lock()
	agentMetrics[metrics.AgentID] = &metrics
	mu.Unlock()

	// Create multi-agent data and broadcast
	multiData := createMultiAgentData()
	
	// Send the new metrics to the broadcast channel (non-blocking)
	select {
	case broadcast <- multiData:
		log.Printf("Received metrics from agent %s (%s)", metrics.AgentID, metrics.Hostname)
	default:
		log.Println("Broadcast channel full, dropping metrics")
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok", 
		"agent_id": metrics.AgentID,
		"timestamp": metrics.Timestamp,
	})
}

// handleGetAgents returns list of all agents
func handleGetAgents(c *gin.Context) {
	mu.RLock()
	summary := make([]*AgentSummary, 0, len(agentMetrics))
	for _, metrics := range agentMetrics {
		isOnline := time.Since(metrics.LastSeen) < agentTimeout
		summary = append(summary, &AgentSummary{
			AgentID:     metrics.AgentID,
			Hostname:    metrics.Hostname,
			LastSeen:    metrics.LastSeen,
			IsOnline:    isOnline,
			CPUUsage:    metrics.CPUUsage,
			MemoryUsage: metrics.Memory.UsedPercent,
			DiskUsage:   metrics.Disk.UsedPercent,
			Uptime:      metrics.Uptime,
		})
	}
	mu.RUnlock()
	
	c.JSON(http.StatusOK, summary)
}

// handleGetAgent returns specific agent data
func handleGetAgent(c *gin.Context) {
	agentID := c.Param("agentId")
	
	mu.RLock()
	metrics, exists := agentMetrics[agentID]
	mu.RUnlock()
	
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}
	
	c.JSON(http.StatusOK, metrics)
}

// createMultiAgentData creates the data structure for all agents
func createMultiAgentData() *MultiAgentData {
	mu.RLock()
	defer mu.RUnlock()
	
	// Create a copy of all agent metrics
	agents := make(map[string]*Metrics, len(agentMetrics))
	summary := make([]*AgentSummary, 0, len(agentMetrics))
	
	for agentID, metrics := range agentMetrics {
		// Deep copy metrics to avoid race conditions
		metricsCopy := *metrics
		agents[agentID] = &metricsCopy
		
		// Create summary
		isOnline := time.Since(metrics.LastSeen) < agentTimeout
		summary = append(summary, &AgentSummary{
			AgentID:     metrics.AgentID,
			Hostname:    metrics.Hostname,
			LastSeen:    metrics.LastSeen,
			IsOnline:    isOnline,
			CPUUsage:    metrics.CPUUsage,
			MemoryUsage: metrics.Memory.UsedPercent,
			DiskUsage:   metrics.Disk.UsedPercent,
			Uptime:      metrics.Uptime,
		})
	}
	
	return &MultiAgentData{
		Agents:    agents,
		Summary:   summary,
		Timestamp: time.Now(),
	}
}

// agentCleanupRoutine removes stale agents
func agentCleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mu.Lock()
		for agentID, metrics := range agentMetrics {
			if time.Since(metrics.LastSeen) > 10*time.Minute {
				log.Printf("Removing stale agent: %s", agentID)
				delete(agentMetrics, agentID)
			}
		}
		mu.Unlock()
	}
}

// handleWebSocket handles new WebSocket connections from the frontend.
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer func() {
		conn.Close()
		delete(clients, conn)
		log.Println("Client disconnected")
	}()

	// Set ping/pong handlers for connection health
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Register new client.
	clients[conn] = true
	log.Println("New WebSocket client connected")

	// Send the current multi-agent data to the new client immediately.
	multiData := createMultiAgentData()
	if err := conn.WriteJSON(multiData); err != nil {
		log.Printf("Error sending initial metrics: %v", err)
		return
	}

	// Start ping ticker
	pingTicker := time.NewTicker(54 * time.Second)
	defer pingTicker.Stop()

	// Keep the connection open and handle ping/pong
	for {
		select {
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		default:
			// Read messages from client (mainly for pong responses)
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}
		}
	}
}

// handleBroadcast listens on the broadcast channel and sends data to all clients.
func handleBroadcast() {
	for {
		multiData := <-broadcast
		// Send to all connected clients.
		for client := range clients {
			client.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.WriteJSON(multiData); err != nil {
				log.Printf("Error writing to client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
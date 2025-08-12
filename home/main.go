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
	MemoryMB      float64 `json:"memory_mb"`
	Command       string  `json:"command"`
}

// DockerContainerStat holds Docker container statistics
type DockerContainerStat struct {
	ContainerID   string `json:"container_id"`
	Name          string `json:"name"`
	CPUPercent    string `json:"cpu_percent"`
	MemoryUsage   string `json:"memory_usage"`
	MemoryLimit   string `json:"memory_limit"`
	MemoryPercent string `json:"memory_percent"`
	NetworkIO     string `json:"network_io"`
	BlockIO       string `json:"block_io"`
	PIDs          string `json:"pids"`
}

// SystemInfo holds additional system information
type SystemInfo struct {
	TotalProcesses  int    `json:"total_processes"`
	DockerAvailable bool   `json:"docker_available"`
	KernelVersion   string `json:"kernel_version"`
	OSRelease       string `json:"os_release"`
	Architecture    string `json:"architecture"`
}

// AgentInfo holds information about the agent itself
type AgentInfo struct {
	Version       string            `json:"version"`
	GoVersion     string            `json:"go_version"`
	NumGoroutines int               `json:"num_goroutines"`
	MemStats      map[string]uint64 `json:"mem_stats"`
	StartTime     time.Time         `json:"start_time"`
}

// Metrics holds all the system metrics we want to collect.
type Metrics struct {
	AgentID     string                   `json:"agent_id"`
	Hostname    string                   `json:"hostname"`
	Uptime      uint64                   `json:"uptime"`
	CPUUsage    float64                  `json:"cpu_usage"`
	Memory      *mem.VirtualMemoryStat   `json:"memory"`
	Disk        *disk.UsageStat          `json:"disk"`
	Network     []net.IOCountersStat     `json:"network"`
	Processes   []*ProcessInfo           `json:"processes"`
	DockerStats []*DockerContainerStat   `json:"docker_stats"`
	AgentInfo   *AgentInfo               `json:"agent_info"`
	SystemInfo  *SystemInfo              `json:"system_info"`
	Timestamp   time.Time                `json:"timestamp"`
	LastSeen    time.Time                `json:"last_seen"`
}

// AgentSummary provides overview information about an agent
type AgentSummary struct {
	AgentID       string    `json:"agent_id"`
	Hostname      string    `json:"hostname"`
	LastSeen      time.Time `json:"last_seen"`
	IsOnline      bool      `json:"is_online"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsage   float64   `json:"memory_usage"`
	DiskUsage     float64   `json:"disk_usage"`
	Uptime        uint64    `json:"uptime"`
	ProcessCount  int       `json:"process_count"`
	DockerCount   int       `json:"docker_count"`
}

// MultiAgentData contains data for all agents
type MultiAgentData struct {
	Agents    map[string]*Metrics `json:"agents"`
	Summary   []*AgentSummary     `json:"summary"`
	Timestamp time.Time           `json:"timestamp"`
}

var (
	// upgrader is used to upgrade HTTP connections to WebSocket connections.
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
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
	broadcast = make(chan *MultiAgentData, 10)
	// Agent timeout duration
	agentTimeout = 2 * time.Minute
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	api := router.Group("/api")
	{
		api.POST("/metrics", handleMetricsPost)
		api.GET("/ws", handleWebSocket)
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
		api.GET("/agents", handleGetAgents)
		api.GET("/agents/:agentId", handleGetAgent)
	}

	router.Use(staticFileHandler("./web/build"))
	go handleBroadcast()
	go agentCleanupRoutine()

	srv := &http.Server{
		Addr:         ":8085",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Home server starting on :8085")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for client := range clients {
		client.Close()
	}

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

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

func staticFileHandler(root string) gin.HandlerFunc {
	fileServer := http.FileServer(http.Dir(root))
	
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Next()
			return
		}

		path := filepath.Join(root, c.Request.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = filepath.Join(root, "index.html")
		}

		c.Request.URL.Path = strings.TrimPrefix(c.Request.URL.Path, "/")
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

func handleMetricsPost(c *gin.Context) {
	var metrics Metrics
	if err := c.ShouldBindJSON(&metrics); err != nil {
		log.Printf("Invalid JSON from agent: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	metrics.Timestamp = time.Now()
	metrics.LastSeen = time.Now()
	
	if metrics.AgentID == "" {
		metrics.AgentID = metrics.Hostname
	}

	mu.Lock()
	agentMetrics[metrics.AgentID] = &metrics
	mu.Unlock()

	multiData := createMultiAgentData()
	
	select {
	case broadcast <- multiData:
		log.Printf("Received metrics from agent %s (%s) - Processes: %d, Docker: %d", 
			metrics.AgentID, metrics.Hostname, len(metrics.Processes), len(metrics.DockerStats))
	default:
		log.Println("Broadcast channel full, dropping metrics")
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok", 
		"agent_id": metrics.AgentID,
		"timestamp": metrics.Timestamp,
	})
}

func handleGetAgents(c *gin.Context) {
	mu.RLock()
	summary := make([]*AgentSummary, 0, len(agentMetrics))
	for _, metrics := range agentMetrics {
		isOnline := time.Since(metrics.LastSeen) < agentTimeout
		summary = append(summary, &AgentSummary{
			AgentID:      metrics.AgentID,
			Hostname:     metrics.Hostname,
			LastSeen:     metrics.LastSeen,
			IsOnline:     isOnline,
			CPUUsage:     metrics.CPUUsage,
			MemoryUsage:  metrics.Memory.UsedPercent,
			DiskUsage:    metrics.Disk.UsedPercent,
			Uptime:       metrics.Uptime,
			ProcessCount: len(metrics.Processes),
			DockerCount:  len(metrics.DockerStats),
		})
	}
	mu.RUnlock()
	
	c.JSON(http.StatusOK, summary)
}

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

func createMultiAgentData() *MultiAgentData {
	mu.RLock()
	defer mu.RUnlock()
	
	agents := make(map[string]*Metrics, len(agentMetrics))
	summary := make([]*AgentSummary, 0, len(agentMetrics))
	
	for agentID, metrics := range agentMetrics {
		metricsCopy := *metrics
		agents[agentID] = &metricsCopy
		
		isOnline := time.Since(metrics.LastSeen) < agentTimeout
		summary = append(summary, &AgentSummary{
			AgentID:      metrics.AgentID,
			Hostname:     metrics.Hostname,
			LastSeen:     metrics.LastSeen,
			IsOnline:     isOnline,
			CPUUsage:     metrics.CPUUsage,
			MemoryUsage:  metrics.Memory.UsedPercent,
			DiskUsage:    metrics.Disk.UsedPercent,
			Uptime:       metrics.Uptime,
			ProcessCount: len(metrics.Processes),
			DockerCount:  len(metrics.DockerStats),
		})
	}
	
	return &MultiAgentData{
		Agents:    agents,
		Summary:   summary,
		Timestamp: time.Now(),
	}
}

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

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	clients[conn] = true
	log.Println("New WebSocket client connected")

	multiData := createMultiAgentData()
	if err := conn.WriteJSON(multiData); err != nil {
		log.Printf("Error sending initial metrics: %v", err)
		return
	}

	pingTicker := time.NewTicker(54 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error sending ping: %v", err)
				return
			}
		default:
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

func handleBroadcast() {
	for {
		multiData := <-broadcast
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
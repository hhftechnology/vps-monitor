package models

// ContainerStats represents real-time container resource usage
type ContainerStats struct {
	ContainerID   string  `json:"container_id"`
	Host          string  `json:"host"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx     uint64  `json:"network_rx"`
	NetworkTx     uint64  `json:"network_tx"`
	BlockRead     uint64  `json:"block_read"`
	BlockWrite    uint64  `json:"block_write"`
	PIDs          uint64  `json:"pids"`
	Timestamp     int64   `json:"timestamp"`
}

type HistoricalAverages struct {
	CPU1h     float64 `json:"cpu_1h"`
	Memory1h  float64 `json:"memory_1h"`
	CPU12h    float64 `json:"cpu_12h"`
	Memory12h float64 `json:"memory_12h"`
	HasData   bool    `json:"has_data"`
}

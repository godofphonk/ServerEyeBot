package domain

import "time"

// MetricsResponse represents the response from /api/servers/by-key/{key}/metrics
type MetricsResponse struct {
	Metrics ServerMetrics `json:"metrics"`
}

// ServerMetrics represents all server metrics
type ServerMetrics struct {
	CPU                float64            `json:"cpu"`
	CPUUsage           CPUUsageDetails    `json:"cpu_usage"`
	Memory             float64            `json:"memory"`
	MemoryDetails      MemoryDetails      `json:"memory_details"`
	Disk               float64            `json:"disk"`
	DiskDetails        []DiskDetails      `json:"disk_details"`
	Network            float64            `json:"network"`
	NetworkDetails     NetworkDetails     `json:"network_details"`
	TemperatureDetails TemperatureDetails `json:"temperature_details"`
	SystemDetails      SystemDetails      `json:"system_details"`
}

// CPUUsageDetails represents detailed CPU usage information
type CPUUsageDetails struct {
	UsageTotal  float64     `json:"usage_total"`
	UsageUser   float64     `json:"usage_user"`
	UsageSystem float64     `json:"usage_system"`
	UsageIdle   float64     `json:"usage_idle"`
	LoadAverage LoadAverage `json:"load_average"`
	Cores       int         `json:"cores"`
	Frequency   float64     `json:"frequency"`
}

// LoadAverage represents system load average
type LoadAverage struct {
	Load1min  float64 `json:"load_1min"`
	Load5min  float64 `json:"load_5min"`
	Load15min float64 `json:"load_15min"`
}

// MemoryDetails represents detailed memory information
type MemoryDetails struct {
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	FreeGB      float64 `json:"free_gb"`
	UsedPercent float64 `json:"used_percent"`
}

// DiskDetails represents disk information for a single filesystem
type DiskDetails struct {
	Path        string  `json:"path"`
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	FreeGB      float64 `json:"free_gb"`
	UsedPercent float64 `json:"used_percent"`
	Filesystem  string  `json:"filesystem"`
}

// NetworkDetails represents detailed network information
type NetworkDetails struct {
	Interfaces  []NetworkInterface `json:"interfaces"`
	TotalRxMbps float64            `json:"total_rx_mbps"`
	TotalTxMbps float64            `json:"total_tx_mbps"`
}

// NetworkInterface represents a network interface with traffic stats
type NetworkInterfaceExtended struct {
	Name   string  `json:"name"`
	RxMbps float64 `json:"rx_mbps"`
	TxMbps float64 `json:"tx_mbps"`
	Status string  `json:"status"`
}

// TemperatureDetails represents temperature information
type TemperatureDetails struct {
	CPUTemperature     float64 `json:"cpu_temperature"`
	GPUTemperature     float64 `json:"gpu_temperature"`
	SystemTemperature  float64 `json:"system_temperature"`
	HighestTemperature float64 `json:"highest_temperature"`
	TemperatureUnit    string  `json:"temperature_unit"`
}

// SystemDetails represents detailed system information
type SystemDetails struct {
	Hostname          string `json:"hostname"`
	OS                string `json:"os"`
	Kernel            string `json:"kernel"`
	Architecture      string `json:"architecture"`
	UptimeSeconds     int    `json:"uptime_seconds"`
	UptimeHuman       string `json:"uptime_human"`
	ProcessesTotal    int    `json:"processes_total"`
	ProcessesRunning  int    `json:"processes_running"`
	ProcessesSleeping int    `json:"processes_sleeping"`
}

// MetricsCache represents cached metrics with TTL
type MetricsCache struct {
	ServerKey string
	Metrics   *MetricsResponse
	ExpiresAt time.Time
}

// MetricsFormatter defines interface for formatting metrics for display
type MetricsFormatter interface {
	FormatCPU(metrics *ServerMetrics) string
	FormatMemory(metrics *ServerMetrics) string
	FormatDisk(metrics *ServerMetrics) string
	FormatTemperature(metrics *ServerMetrics) string
	FormatNetwork(metrics *ServerMetrics) string
	FormatSystem(metrics *ServerMetrics) string
	FormatAll(metrics *ServerMetrics) string
}

// ServerMetricsService defines interface for working with server metrics
type ServerMetricsService interface {
	GetServerMetrics(serverKey string) (*MetricsResponse, error)
	FormatCPU(metrics *ServerMetrics) string
	FormatMemory(metrics *ServerMetrics) string
	FormatDisk(metrics *ServerMetrics) string
	FormatTemperature(metrics *ServerMetrics) string
	FormatNetwork(metrics *ServerMetrics) string
	FormatSystem(metrics *ServerMetrics) string
	FormatAll(metrics *ServerMetrics) string
}

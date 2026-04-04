package domain

import "time"

// MetricsResponse represents the response from /api/servers/by-key/{key}/metrics
type MetricsResponse struct {
	Metrics   NewServerMetrics `json:"metrics"`
	ServerID  string           `json:"server_id"`
	ServerKey string           `json:"server_key"`
}

// LegacyMetricsResponse represents the legacy response format for compatibility
type LegacyMetricsResponse struct {
	Metrics   ServerMetrics `json:"metrics"`
	ServerID  string        `json:"server_id"`
	ServerKey string        `json:"server_key"`
}

// ServerMetrics represents all server metrics (legacy compatibility)
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

// NewServerMetrics represents the new API metrics structure
type NewServerMetrics struct {
	CPUPercent         float64               `json:"cpu_percent"`
	DiskPercent        float64               `json:"disk_percent"`
	LoadAverage        LoadAverageNew        `json:"load_average"`
	MemoryPercent      float64               `json:"memory_percent"`
	MemoryDetails      NewMemoryDetails      `json:"memory_details"`
	DiskDetails        []NewDiskDetails      `json:"disk_details"`
	NetworkDetails     NewNetworkDetails     `json:"network_details"`
	NetworkMbps        float64               `json:"network_mbps"`
	ProcessesRunning   int                   `json:"processes_running"`
	ProcessesSleeping  int                   `json:"processes_sleeping"`
	ProcessesTotal     int                   `json:"processes_total"`
	TemperatureCelsius float64               `json:"temperature_celsius"`
	Temperatures       NewTemperatureDetails `json:"temperatures"`
	Timestamp          string                `json:"timestamp"`
	UptimeSeconds      int                   `json:"uptime_seconds"`
}

// LoadAverageNew represents new load average structure
type LoadAverageNew struct {
	Min1  float64 `json:"1m"`
	Min5  float64 `json:"5m"`
	Min15 float64 `json:"15m"`
}

// NewMemoryDetails represents new memory details structure
type NewMemoryDetails struct {
	AvailableGB float64 `json:"available_gb"`
	BuffersGB   float64 `json:"buffers_gb"`
	CachedGB    float64 `json:"cached_gb"`
	FreeGB      float64 `json:"free_gb"`
	UsedGB      float64 `json:"used_gb"`
}

// NewDiskDetails represents new disk details structure
type NewDiskDetails struct {
	FreeGB      float64 `json:"free_gb"`
	Path        string  `json:"path"`
	UsedGB      float64 `json:"used_gb"`
	UsedPercent int     `json:"used_percent"`
}

// NewNetworkDetails represents new network details structure
type NewNetworkDetails struct {
	TotalRxMbps float64 `json:"total_rx_mbps"`
	TotalTxMbps float64 `json:"total_tx_mbps"`
}

// NewTemperatureDetails represents new temperature details structure
type NewTemperatureDetails struct {
	CPU     float64              `json:"cpu"`
	GPU     float64              `json:"gpu"`
	Highest float64              `json:"highest"`
	Storage []StorageTemperature `json:"storage"`
}

// StorageTemperature represents storage device temperature
type StorageTemperature struct {
	Device      string  `json:"device"`
	Type        string  `json:"type"`
	Temperature float64 `json:"temperature"`
}

// StaticInfoResponse represents response from /api/servers/by-key/{key}/static-info
type StaticInfoResponse struct {
	ServerInfo        ServerInfo             `json:"server_info"`
	HardwareInfo      HardwareInfo           `json:"hardware_info"`
	MotherboardInfo   MotherboardInfo        `json:"motherboard_info"`
	NetworkInterfaces []NetworkInterfaceInfo `json:"network_interfaces"`
	DiskInfo          []DiskInfo             `json:"disk_info"`
}

// ServerInfo represents basic server information
type ServerInfo struct {
	ServerID     string `json:"server_id"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	OSVersion    string `json:"os_version"`
	Kernel       string `json:"kernel"`
	Architecture string `json:"architecture"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// HardwareInfo represents hardware information
type HardwareInfo struct {
	ServerID        string  `json:"server_id"`
	CPUModel        string  `json:"cpu_model"`
	CPUCores        int     `json:"cpu_cores"`
	CPUThreads      int     `json:"cpu_threads"`
	CPUFrequencyMHz float64 `json:"cpu_frequency_mhz"`
	GPUModel        string  `json:"gpu_model"`
	GPUDriver       string  `json:"gpu_driver"`
	GPUMemoryGB     int     `json:"gpu_memory_gb"`
	TotalMemoryGB   int     `json:"total_memory_gb"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// MotherboardInfo represents motherboard information
type MotherboardInfo struct {
	ServerID     string `json:"server_id"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	BIOSDate     string `json:"bios_date"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// NetworkInterfaceInfo represents network interface information for API
type NetworkInterfaceInfo struct {
	ID            int    `json:"id"`
	ServerID      string `json:"server_id"`
	InterfaceName string `json:"interface_name"`
	MACAddress    string `json:"mac_address"`
	InterfaceType string `json:"interface_type"`
	SpeedMbps     int    `json:"speed_mbps"`
	Vendor        string `json:"vendor"`
	Driver        string `json:"driver"`
	IsPhysical    bool   `json:"is_physical"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	ID            int    `json:"id"`
	ServerID      string `json:"server_id"`
	DeviceName    string `json:"device_name"`
	Model         string `json:"model"`
	SerialNumber  string `json:"serial_number"`
	SizeGB        int    `json:"size_gb"`
	DiskType      string `json:"disk_type"`
	InterfaceType string `json:"interface_type"`
	Filesystem    string `json:"filesystem"`
	MountPoint    string `json:"mount_point"`
	IsSystemDisk  bool   `json:"is_system_disk"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// ServerStatusResponse represents response from /api/servers/by-key/{key}/status
type ServerStatusResponse struct {
	AgentVersion string `json:"agent_version"`
	LastSeen     string `json:"last_seen"`
	Online       bool   `json:"online"`
	ServerID     string `json:"server_id"`
	ServerKey    string `json:"server_key"`
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
	Metrics   *LegacyMetricsResponse
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
	GetServerMetrics(serverKey string) (*LegacyMetricsResponse, error)
	FormatCPU(metrics *ServerMetrics) string
	FormatMemory(metrics *ServerMetrics) string
	FormatDisk(metrics *ServerMetrics) string
	FormatTemperature(metrics *ServerMetrics) string
	FormatNetwork(metrics *ServerMetrics) string
	FormatSystem(metrics *ServerMetrics) string
	FormatAll(metrics *ServerMetrics) string
}

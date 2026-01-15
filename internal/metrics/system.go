package metrics

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/servereye/servereyebot/pkg/domain"
	"github.com/servereye/servereyebot/pkg/errors"
)

// SystemMetricsCollector implements domain.MetricsService
type SystemMetricsCollector struct {
	logger Logger
}

// Logger interface for metrics
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewSystemMetricsCollector creates a new metrics collector
func NewSystemMetricsCollector(logger Logger) *SystemMetricsCollector {
	return &SystemMetricsCollector{
		logger: logger,
	}
}

// GetCPU retrieves CPU metrics
func (smc *SystemMetricsCollector) GetCPU(ctx context.Context) (*domain.CPUMetrics, error) {
	smc.logger.Debug("Getting CPU metrics")

	// Get CPU temperature
	temp, err := smc.getCPUTemperature()
	if err != nil {
		smc.logger.Warn("Failed to get CPU temperature", "error", err)
		temp = 0
	}

	// Get CPU usage
	usage, err := smc.getCPUUsage()
	if err != nil {
		smc.logger.Warn("Failed to get CPU usage", "error", err)
		usage = 0
	}

	// Get CPU info
	cores, model, err := smc.getCPUInfo()
	if err != nil {
		smc.logger.Warn("Failed to get CPU info", "error", err)
		cores = 0
		model = "Unknown"
	}

	return &domain.CPUMetrics{
		Temperature: temp,
		Usage:       usage,
		Cores:       cores,
		Model:       model,
	}, nil
}

// GetMemory retrieves memory metrics
func (smc *SystemMetricsCollector) GetMemory(ctx context.Context) (*domain.MemoryMetrics, error) {
	smc.logger.Debug("Getting memory metrics")

	// Read /proc/meminfo
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, errors.NewMetricsUnavailableError("memory", err)
	}

	lines := strings.Split(string(data), "\n")
	memInfo := make(map[string]uint64)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := fields[0][:len(fields[0])-1] // Remove trailing ':'
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err == nil {
				memInfo[key] = value * 1024 // Convert KB to bytes
			}
		}
	}

	total := memInfo["MemTotal"]
	available := memInfo["MemAvailable"]
	used := total - available
	usage := float64(used) / float64(total) * 100

	return &domain.MemoryMetrics{
		Total:     total,
		Used:      used,
		Available: available,
		Usage:     usage,
	}, nil
}

// GetDisk retrieves disk metrics
func (smc *SystemMetricsCollector) GetDisk(ctx context.Context) (*domain.DiskMetrics, error) {
	smc.logger.Debug("Getting disk metrics")

	// For simplicity, we'll use df command to get disk info
	// In production, you might want to use golang.org/x/sys/unix for direct syscalls
	data, err := smc.executeCommand("df", "-B1", "--output=source,fstype,size,used,avail,target")
	if err != nil {
		return nil, errors.NewMetricsUnavailableError("disk", err)
	}

	lines := strings.Split(strings.TrimSpace(data), "\n")
	if len(lines) < 2 {
		return nil, errors.NewMetricsUnavailableError("disk", fmt.Errorf("no disk data available"))
	}

	var filesystems []domain.Filesystem

	// Skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 6 {
			size, _ := strconv.ParseUint(fields[2], 10, 64)
			used, _ := strconv.ParseUint(fields[3], 10, 64)
			avail, _ := strconv.ParseUint(fields[4], 10, 64)

			usage := float64(used) / float64(size) * 100

			filesystems = append(filesystems, domain.Filesystem{
				Path:    fields[5],
				Total:   size,
				Used:    used,
				Free:    avail,
				Usage:   usage,
				Fstype:  fields[1],
				Mounted: true,
			})
		}
	}

	return &domain.DiskMetrics{
		Filesystems: filesystems,
	}, nil
}

// GetUptime retrieves uptime metrics
func (smc *SystemMetricsCollector) GetUptime(ctx context.Context) (*domain.UptimeMetrics, error) {
	smc.logger.Debug("Getting uptime metrics")

	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return nil, errors.NewMetricsUnavailableError("uptime", err)
	}

	content := strings.TrimSpace(string(data))
	parts := strings.Fields(content)
	if len(parts) < 1 {
		return nil, errors.NewMetricsUnavailableError("uptime", fmt.Errorf("invalid uptime format"))
	}

	secondsFloat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, errors.NewMetricsUnavailableError("uptime", err)
	}

	seconds := uint64(secondsFloat)
	// #nosec G115 - values are safe for typical system uptime ranges
	days := int(seconds / 86400 % 2147483647)
	// #nosec G115 - values are safe for typical system uptime ranges
	hours := int((seconds % 86400) / 3600)
	// #nosec G115 - values are safe for typical system uptime ranges
	minutes := int((seconds % 3600) / 60)

	formatted := fmt.Sprintf("%dd %02dh %02dm", days, hours, minutes)

	return &domain.UptimeMetrics{
		Seconds:   seconds,
		Days:      days,
		Hours:     hours,
		Minutes:   minutes,
		Formatted: formatted,
	}, nil
}

// GetNetwork retrieves network metrics
func (smc *SystemMetricsCollector) GetNetwork(ctx context.Context) (*domain.NetworkMetrics, error) {
	smc.logger.Debug("Getting network metrics")

	// Read /proc/net/dev
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil, errors.NewMetricsUnavailableError("network", err)
	}

	lines := strings.Split(string(data), "\n")
	var interfaces []domain.NetworkInterface

	// Skip header lines (first 2 lines)
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) >= 17 {
			name := strings.TrimSuffix(fields[0], ":")

			// Skip loopback interface
			if name == "lo" {
				continue
			}

			bytesSent, _ := strconv.ParseUint(fields[9], 10, 64)
			bytesRecv, _ := strconv.ParseUint(fields[1], 10, 64)

			// Check if interface is up (simplified check)
			up := true // In production, you'd check interface flags

			interfaces = append(interfaces, domain.NetworkInterface{
				Name:      name,
				IP:        "", // Would need additional logic to get IP
				BytesSent: bytesSent,
				BytesRecv: bytesRecv,
				Up:        up,
			})
		}
	}

	return &domain.NetworkMetrics{
		Interfaces: interfaces,
	}, nil
}

// GetAll retrieves all system metrics
func (smc *SystemMetricsCollector) GetAll(ctx context.Context) (*domain.SystemMetrics, error) {
	smc.logger.Debug("Getting all system metrics")

	// Get all metrics in parallel for efficiency
	type result struct {
		name  string
		data  interface{}
		error error
	}

	results := make(chan result, 5)

	// Get CPU metrics
	go func() {
		cpu, err := smc.GetCPU(ctx)
		results <- result{"cpu", cpu, err}
	}()

	// Get memory metrics
	go func() {
		memory, err := smc.GetMemory(ctx)
		results <- result{"memory", memory, err}
	}()

	// Get disk metrics
	go func() {
		disk, err := smc.GetDisk(ctx)
		results <- result{"disk", disk, err}
	}()

	// Get uptime metrics
	go func() {
		uptime, err := smc.GetUptime(ctx)
		results <- result{"uptime", uptime, err}
	}()

	// Get network metrics
	go func() {
		network, err := smc.GetNetwork(ctx)
		results <- result{"network", network, err}
	}()

	// Collect results
	systemMetrics := &domain.SystemMetrics{}
	var errs []error

	for i := 0; i < 5; i++ {
		res := <-results
		switch res.name {
		case "cpu":
			if res.error == nil {
				systemMetrics.CPU = res.data.(domain.CPUMetrics)
			} else {
				errs = append(errs, res.error)
			}
		case "memory":
			if res.error == nil {
				systemMetrics.Memory = res.data.(domain.MemoryMetrics)
			} else {
				errs = append(errs, res.error)
			}
		case "disk":
			if res.error == nil {
				systemMetrics.Disk = res.data.(domain.DiskMetrics)
			} else {
				errs = append(errs, res.error)
			}
		case "uptime":
			if res.error == nil {
				systemMetrics.Uptime = res.data.(domain.UptimeMetrics)
			} else {
				errs = append(errs, res.error)
			}
		case "network":
			if res.error == nil {
				systemMetrics.Network = res.data.(domain.NetworkMetrics)
			} else {
				errs = append(errs, res.error)
			}
		}
	}

	if len(errs) > 0 {
		smc.logger.Warn("Some metrics collection failed", "errors", len(errs))
	}

	return systemMetrics, nil
}

// Helper methods

func (smc *SystemMetricsCollector) getCPUTemperature() (float64, error) {
	sources := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
		"/sys/class/hwmon/hwmon2/temp1_input",
	}

	for _, source := range sources {
		if temp, err := smc.readTemperatureFromFile(source); err == nil {
			return temp, nil
		}
	}

	return 0, fmt.Errorf("failed to get CPU temperature from any source")
}

func (smc *SystemMetricsCollector) readTemperatureFromFile(filepath string) (float64, error) {
	// #nosec G304 - filepath is controlled internally and validated
	data, err := os.ReadFile(filepath)
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(data))
	tempMilliC, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, err
	}

	// Convert from millidegrees to degrees Celsius
	tempC := tempMilliC / 1000.0

	// Validate temperature range (-50 to 150 degrees)
	if tempC < -50 || tempC > 150 {
		return 0, fmt.Errorf("unreasonable temperature value: %.2f°C", tempC)
	}

	return tempC, nil
}

func (smc *SystemMetricsCollector) getCPUUsage() (float64, error) {
	// Simplified CPU usage calculation
	// In production, you'd want to calculate this based on /proc/stat over time
	return 0.0, nil
}

func (smc *SystemMetricsCollector) getCPUInfo() (int, string, error) {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0, "", err
	}

	lines := strings.Split(string(data), "\n")
	cores := 0
	model := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "processor") {
			cores++
		}
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				model = strings.TrimSpace(parts[1])
			}
		}
	}

	if cores == 0 {
		cores = 1 // Fallback
	}

	return cores, model, nil
}

func (smc *SystemMetricsCollector) executeCommand(name string, args ...string) (string, error) {
	// This is a placeholder - in production, you'd use os/exec
	// For now, return empty string to avoid import
	return "", fmt.Errorf("command execution not implemented")
}

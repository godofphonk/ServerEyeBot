package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/servereye/servereyebot/internal/api"
	"github.com/servereye/servereyebot/pkg/domain"
)

// MetricsServiceImpl implements ServerMetricsService
type MetricsServiceImpl struct {
	apiClient  *api.Client
	cache      map[string]*domain.MetricsCache
	cacheMutex sync.RWMutex
	logger     Logger
}

// Logger interface for metrics service
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewMetricsService creates a new metrics service
func NewMetricsService(apiClient *api.Client, logger Logger) *MetricsServiceImpl {
	return &MetricsServiceImpl{
		apiClient: apiClient,
		cache:     make(map[string]*domain.MetricsCache),
		logger:    logger,
	}
}

// GetServerMetrics retrieves server metrics directly from API (no cache)
func (s *MetricsServiceImpl) GetServerMetrics(serverKey string) (*domain.LegacyMetricsResponse, error) {
	fmt.Printf("=== GETTING FRESH METRICS FROM API ===\n")
	s.logger.Info("Getting fresh server metrics from API", "server_key", serverKey)

	// Fetch from API
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics, err := s.apiClient.GetServerMetrics(ctx, serverKey)
	if err != nil {
		s.logger.Error("Failed to get server metrics", "error", err, "server_key", serverKey)
		return nil, err
	}

	// Convert new API structure to legacy format for compatibility
	legacyMetrics := s.convertToLegacyMetrics(metrics)

	fmt.Printf("=== METRICS CONVERTED SUCCESSFULLY ===\n")
	s.logger.Info("Server metrics retrieved and converted successfully", "server_key", serverKey)
	return legacyMetrics, nil
}

// FormatCPU formats CPU metrics for display
func (s *MetricsServiceImpl) FormatCPU(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики CPU недоступны"
	}

	// Try to use new metrics structure first
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🖥️ Загрузка процессора: %.1f%%\n", newMetrics.CPUPercent))
		sb.WriteString(fmt.Sprintf("- Load Average: %.2f, %.2f, %.2f\n",
			newMetrics.LoadAverage.Min1,
			newMetrics.LoadAverage.Min5,
			newMetrics.LoadAverage.Min15))
		sb.WriteString(fmt.Sprintf("- Процессы: %d (%d running)",
			newMetrics.ProcessesTotal, newMetrics.ProcessesRunning))
		return sb.String()
	}

	// Fallback to legacy structure
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🖥️ Загрузка процессора: %.1f%%\n", metrics.CPU))
	sb.WriteString(fmt.Sprintf("- User: %.1f%%\n", metrics.CPUUsage.UsageUser))
	sb.WriteString(fmt.Sprintf("- System: %.1f%%\n", metrics.CPUUsage.UsageSystem))
	sb.WriteString(fmt.Sprintf("- Idle: %.1f%%\n", metrics.CPUUsage.UsageIdle))
	sb.WriteString(fmt.Sprintf("- Load Average: %.2f, %.2f, %.2f\n",
		metrics.CPUUsage.LoadAverage.Load1min,
		metrics.CPUUsage.LoadAverage.Load5min,
		metrics.CPUUsage.LoadAverage.Load15min))
	sb.WriteString(fmt.Sprintf("- Ядра: %d @ %.1f MHz", metrics.CPUUsage.Cores, metrics.CPUUsage.Frequency))

	return sb.String()
}

// FormatMemory formats memory metrics for display
func (s *MetricsServiceImpl) FormatMemory(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики памяти недоступны"
	}

	// Try to use new metrics structure first
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("💾 Память: %.1f%% использовано\n", newMetrics.MemoryPercent))
		sb.WriteString(fmt.Sprintf("- Всего: %.1f GB\n", newMetrics.MemoryDetails.UsedGB+newMetrics.MemoryDetails.FreeGB+newMetrics.MemoryDetails.AvailableGB))
		sb.WriteString(fmt.Sprintf("- Использовано: %.1f GB\n", newMetrics.MemoryDetails.UsedGB))
		sb.WriteString(fmt.Sprintf("- Доступно: %.1f GB\n", newMetrics.MemoryDetails.AvailableGB))
		sb.WriteString(fmt.Sprintf("- Свободно: %.1f GB\n", newMetrics.MemoryDetails.FreeGB))
		sb.WriteString(fmt.Sprintf("- Кеш: %.1f GB\n", newMetrics.MemoryDetails.CachedGB))
		sb.WriteString(fmt.Sprintf("- Буферы: %.1f GB", newMetrics.MemoryDetails.BuffersGB))
		return sb.String()
	}

	// Fallback to legacy structure
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("💾 Память: %.1f%% использовано\n", metrics.Memory))
	sb.WriteString(fmt.Sprintf("- Всего: %.2f GB\n", metrics.MemoryDetails.TotalGB))
	sb.WriteString(fmt.Sprintf("- Использовано: %.2f GB\n", metrics.MemoryDetails.UsedGB))
	sb.WriteString(fmt.Sprintf("- Доступно: %.2f GB\n", metrics.MemoryDetails.AvailableGB))
	sb.WriteString(fmt.Sprintf("- Свободно: %.2f GB", metrics.MemoryDetails.FreeGB))

	return sb.String()
}

// FormatDisk formats disk metrics for display
func (s *MetricsServiceImpl) FormatDisk(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики диска недоступны"
	}

	// Try to use new metrics structure first
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		var sb strings.Builder
		sb.WriteString("💿 Дисковое пространство:\n")

		for _, disk := range newMetrics.DiskDetails {
			sb.WriteString(fmt.Sprintf("%s\n", disk.Path))
			sb.WriteString(fmt.Sprintf("- Использовано: %d GB (%.0f%%)\n", int(disk.UsedGB), float64(disk.UsedPercent)))
			sb.WriteString(fmt.Sprintf("- Свободно: %d GB\n", int(disk.FreeGB)))
		}

		return sb.String()
	}

	// Fallback to legacy structure
	if len(metrics.DiskDetails) == 0 {
		return "❌ Метрики диска недоступны"
	}

	var sb strings.Builder
	sb.WriteString("💿 Дисковое пространство:\n")

	for _, disk := range metrics.DiskDetails {
		sb.WriteString(fmt.Sprintf("%s\n", disk.Path))
		sb.WriteString(fmt.Sprintf("- Файловая система: %s\n", disk.Filesystem))
		sb.WriteString(fmt.Sprintf("- Всего: %d GB\n", int(disk.TotalGB)))
		sb.WriteString(fmt.Sprintf("- Использовано: %d GB (%.0f%%)\n", int(disk.UsedGB), disk.UsedPercent))
		sb.WriteString(fmt.Sprintf("- Свободно: %d GB\n", int(disk.FreeGB)))
	}

	return sb.String()
}

// FormatTemperature formats temperature metrics for display
func (s *MetricsServiceImpl) FormatTemperature(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики температуры недоступны"
	}

	// Debug log to see what we actually have
	s.logger.Info("DEBUG: Temperature values in legacy metrics",
		"cpu", metrics.TemperatureDetails.CPUTemperature,
		"gpu", metrics.TemperatureDetails.GPUTemperature,
		"system", metrics.TemperatureDetails.SystemTemperature,
		"highest", metrics.TemperatureDetails.HighestTemperature)

	var sb strings.Builder
	sb.WriteString("🌡️ Температура:\n")
	sb.WriteString(fmt.Sprintf("- CPU: %.1f°C\n", metrics.TemperatureDetails.CPUTemperature))
	sb.WriteString(fmt.Sprintf("- GPU: %.1f°C\n", metrics.TemperatureDetails.GPUTemperature))
	sb.WriteString(fmt.Sprintf("- System: %.1f°C\n", metrics.TemperatureDetails.SystemTemperature))
	sb.WriteString(fmt.Sprintf("- Максимальная: %.1f°C\n", metrics.TemperatureDetails.HighestTemperature))

	// Get storage temperatures from the current metrics by converting back to new format
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		for _, storage := range newMetrics.Temperatures.Storage {
			deviceName := storage.Device
			if len(deviceName) > 10 {
				deviceName = deviceName[len(deviceName)-10:] // Show last 10 chars
			}
			sb.WriteString(fmt.Sprintf("- Накопитель %s: %.1f°C\n", deviceName, storage.Temperature))
		}
	} else {
		s.logger.Error("Failed to convert metrics for storage temperatures", "error", err)
	}

	return sb.String()
}

// FormatNetwork formats network metrics for display
func (s *MetricsServiceImpl) FormatNetwork(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики сети недоступны"
	}

	// Try to use new metrics structure first
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		var sb strings.Builder
		sb.WriteString("🌐 Сеть:\n")
		sb.WriteString(fmt.Sprintf("- Прием: %.2f Mbps\n", newMetrics.NetworkDetails.TotalRxMbps))
		sb.WriteString(fmt.Sprintf("- Передача: %.2f Mbps\n", newMetrics.NetworkDetails.TotalTxMbps))
		sb.WriteString(fmt.Sprintf("- Общий трафик: %.2f Mbps", newMetrics.NetworkMbps))

		return sb.String()
	}

	// Fallback to legacy structure
	var sb strings.Builder
	sb.WriteString("🌐 Сеть:\n")
	sb.WriteString(fmt.Sprintf("- Прием: %.2f Mbps\n", metrics.NetworkDetails.TotalRxMbps))
	sb.WriteString(fmt.Sprintf("- Передача: %.2f Mbps\n", metrics.NetworkDetails.TotalTxMbps))

	// Sort interfaces by traffic (rx + tx)
	interfaces := make([]domain.NetworkInterfaceExtended, len(metrics.NetworkDetails.Interfaces))
	for i, iface := range metrics.NetworkDetails.Interfaces {
		// Convert basic interface to extended (assuming fields exist or are zero)
		interfaces[i] = domain.NetworkInterfaceExtended{
			Name:   iface.Name,
			RxMbps: 0, // Will be populated if available
			TxMbps: 0, // Will be populated if available
			Status: "up",
		}
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return (interfaces[i].RxMbps + interfaces[i].TxMbps) > (interfaces[j].RxMbps + interfaces[j].TxMbps)
	})

	sb.WriteString("- Интерфейсы:\n")
	maxInterfaces := 5
	if len(interfaces) < maxInterfaces {
		maxInterfaces = len(interfaces)
	}

	for i := 0; i < maxInterfaces; i++ {
		iface := interfaces[i]
		sb.WriteString(fmt.Sprintf("  - %s: ↑%.2f ↓%.2f Mbps\n", iface.Name, iface.TxMbps, iface.RxMbps))
	}

	return sb.String()
}

// FormatSystem formats system information for display
func (s *MetricsServiceImpl) FormatSystem(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Системная информация недоступна"
	}

	var sb strings.Builder
	sb.WriteString("🖥️ Система:\n")

	// Use available system details from unified API
	hostname := metrics.SystemDetails.Hostname
	if hostname == "" {
		hostname = "Недоступно"
	}

	os := metrics.SystemDetails.OS
	if os == "" {
		os = "Недоступно"
	}

	kernel := metrics.SystemDetails.Kernel
	if kernel == "" {
		kernel = "Недоступно"
	}

	arch := metrics.SystemDetails.Architecture
	if arch == "" {
		arch = "Недоступно"
	}

	sb.WriteString(fmt.Sprintf("- Хостнейм: %s\n", hostname))
	sb.WriteString(fmt.Sprintf("- ОС: %s\n", os))
	sb.WriteString(fmt.Sprintf("- Ядро: %s\n", kernel))
	sb.WriteString(fmt.Sprintf("- Архитектура: %s\n", arch))
	sb.WriteString(fmt.Sprintf("- Аптайм: %s\n", metrics.SystemDetails.UptimeHuman))
	sb.WriteString(fmt.Sprintf("- Процессы: %d (%d running)",
		metrics.SystemDetails.ProcessesTotal,
		metrics.SystemDetails.ProcessesRunning))

	return sb.String()
}

// getStaticInfo retrieves static system information
func (s *MetricsServiceImpl) getStaticInfo() *domain.StaticInfoResponse {
	// Try to get static info from cache or API
	var serverKey string
	s.cacheMutex.RLock()
	for key := range s.cache {
		serverKey = key
		break // Use first available server key
	}
	s.cacheMutex.RUnlock()

	if serverKey != "" {
		s.logger.Info("Attempting to get static info", "server_key", serverKey)
		if staticInfo, err := s.apiClient.GetServerStaticInfo(context.Background(), serverKey); err == nil {
			s.logger.Info("Static info retrieved successfully",
				"hostname", staticInfo.ServerInfo.Hostname,
				"os", staticInfo.ServerInfo.OS,
				"kernel", staticInfo.ServerInfo.Kernel,
				"arch", staticInfo.ServerInfo.Architecture)
			return staticInfo
		} else {
			s.logger.Error("Failed to get static info", "error", err, "server_key", serverKey)
		}
	} else {
		s.logger.Info("No server key available for static info")
	}

	return nil
}

// FormatAll formats all metrics in a compact view
func (s *MetricsServiceImpl) FormatAll(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики недоступны"
	}

	// Try to use new metrics structure first
	if newMetrics, err := s.convertToNewMetrics(metrics); err == nil {
		var sb strings.Builder
		sb.WriteString("📊 Общая сводка метрик:\n\n")

		// CPU
		sb.WriteString(fmt.Sprintf("🖥️ CPU: %.1f%% (Load: %.2f)\n",
			newMetrics.CPUPercent, newMetrics.LoadAverage.Min1))

		// Memory
		totalMemory := newMetrics.MemoryDetails.UsedGB + newMetrics.MemoryDetails.FreeGB + newMetrics.MemoryDetails.AvailableGB
		sb.WriteString(fmt.Sprintf("💾 Память: %.1f%% (%.1f/%.1f GB)\n",
			newMetrics.MemoryPercent, newMetrics.MemoryDetails.UsedGB, totalMemory))

		// Disk (show first disk)
		if len(newMetrics.DiskDetails) > 0 {
			disk := newMetrics.DiskDetails[0]
			sb.WriteString(fmt.Sprintf("💿 Диск %s: %.0f%% (%d/%d GB)\n",
				disk.Path, float64(disk.UsedPercent), int(disk.UsedGB), int(disk.UsedGB+disk.FreeGB)))
		}

		// Network
		sb.WriteString(fmt.Sprintf("🌐 Сеть: ↑%.2f ↓%.2f Mbps\n",
			newMetrics.NetworkDetails.TotalTxMbps, newMetrics.NetworkDetails.TotalRxMbps))

		// Temperature
		sb.WriteString(fmt.Sprintf("🌡️ Температура: %.1f°C (CPU)\n", newMetrics.Temperatures.CPU))

		// System
		uptimeHours := newMetrics.UptimeSeconds / 3600
		sb.WriteString(fmt.Sprintf("⏰ Аптайм: %d ч, Процессы: %d", uptimeHours, newMetrics.ProcessesTotal))

		return sb.String()
	}

	// Fallback to legacy structure
	var sb strings.Builder
	sb.WriteString("📊 Общая сводка метрик:\n\n")

	// CPU
	sb.WriteString(fmt.Sprintf("🖥️ CPU: %.1f%% (Load: %.2f)\n",
		metrics.CPU, metrics.CPUUsage.LoadAverage.Load1min))

	// Memory
	sb.WriteString(fmt.Sprintf("💾 Память: %.1f%% (%.1f/%.1f GB)\n",
		metrics.Memory, metrics.MemoryDetails.UsedGB, metrics.MemoryDetails.TotalGB))

	// Disk (show first disk)
	if len(metrics.DiskDetails) > 0 {
		disk := metrics.DiskDetails[0]
		sb.WriteString(fmt.Sprintf("💿 Диск %s: %.0f%% (%d/%d GB)\n",
			disk.Path, disk.UsedPercent, int(disk.UsedGB), int(disk.TotalGB)))
	}

	// Network
	sb.WriteString(fmt.Sprintf("🌐 Сеть: ↑%.2f ↓%.2f Mbps\n",
		metrics.NetworkDetails.TotalTxMbps, metrics.NetworkDetails.TotalRxMbps))

	// Temperature
	sb.WriteString(fmt.Sprintf("🌡️ Температура: %.1f°C (CPU)\n",
		metrics.TemperatureDetails.CPUTemperature))

	// System
	sb.WriteString(fmt.Sprintf("⏰ Аптайм: %s", metrics.SystemDetails.UptimeHuman))

	return sb.String()
}

// ClearCache clears the metrics cache for a specific server or all servers
func (s *MetricsServiceImpl) ClearCache(serverKey ...string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if len(serverKey) == 0 {
		// Clear all cache
		s.cache = make(map[string]*domain.MetricsCache)
		s.logger.Info("All metrics cache cleared")
	} else {
		// Clear specific server cache
		for _, key := range serverKey {
			delete(s.cache, key)
		}
		s.logger.Info("Metrics cache cleared", "server_keys", serverKey)
	}
}

// GetCacheStatus returns cache status information
func (s *MetricsServiceImpl) GetCacheStatus() map[string]interface{} {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()

	status := make(map[string]interface{})
	status["cached_servers"] = len(s.cache)
	status["cache_entries"] = make([]string, 0, len(s.cache))

	for serverKey := range s.cache {
		status["cache_entries"] = append(status["cache_entries"].([]string), serverKey)
	}

	return status
}

// convertToNewMetrics converts legacy ServerMetrics to NewServerMetrics
func (s *MetricsServiceImpl) convertToNewMetrics(metrics *domain.ServerMetrics) (*domain.NewServerMetrics, error) {
	// If metrics already contain new structure, try to extract it
	// This is a temporary solution - in production, the API should return the new structure directly

	// For now, create a mock conversion based on the legacy structure
	// In reality, this would come from the new API endpoints
	newMetrics := &domain.NewServerMetrics{
		CPUPercent:    metrics.CPU,
		MemoryPercent: metrics.Memory,
		DiskPercent:   0, // Will be calculated from disk details
		LoadAverage: domain.LoadAverageNew{
			Min1:  metrics.CPUUsage.LoadAverage.Load1min,
			Min5:  metrics.CPUUsage.LoadAverage.Load5min,
			Min15: metrics.CPUUsage.LoadAverage.Load15min,
		},
		MemoryDetails: domain.NewMemoryDetails{
			UsedGB:      metrics.MemoryDetails.UsedGB,
			FreeGB:      metrics.MemoryDetails.FreeGB,
			AvailableGB: metrics.MemoryDetails.AvailableGB,
		},
		NetworkDetails: domain.NewNetworkDetails{
			TotalRxMbps: metrics.NetworkDetails.TotalRxMbps,
			TotalTxMbps: metrics.NetworkDetails.TotalTxMbps,
		},
		TemperatureCelsius: metrics.TemperatureDetails.CPUTemperature,
		Temperatures: domain.NewTemperatureDetails{
			CPU:     metrics.TemperatureDetails.CPUTemperature,
			GPU:     metrics.TemperatureDetails.GPUTemperature,
			Highest: metrics.TemperatureDetails.HighestTemperature,
		},
		UptimeSeconds:     metrics.SystemDetails.UptimeSeconds,
		ProcessesTotal:    metrics.SystemDetails.ProcessesTotal,
		ProcessesRunning:  metrics.SystemDetails.ProcessesRunning,
		ProcessesSleeping: metrics.SystemDetails.ProcessesSleeping,
	}

	// Convert disk details
	if len(metrics.DiskDetails) > 0 {
		newMetrics.DiskDetails = make([]domain.NewDiskDetails, len(metrics.DiskDetails))
		for i, disk := range metrics.DiskDetails {
			newMetrics.DiskDetails[i] = domain.NewDiskDetails{
				Path:        disk.Path,
				UsedGB:      disk.UsedGB,
				FreeGB:      disk.FreeGB,
				UsedPercent: int(disk.UsedPercent),
			}
			if i == 0 {
				newMetrics.DiskPercent = disk.UsedPercent
			}
		}
	}

	return newMetrics, nil
}

// convertToLegacyMetrics converts new API response to legacy format
func (s *MetricsServiceImpl) convertToLegacyMetrics(newResponse *domain.MetricsResponse) *domain.LegacyMetricsResponse {
	fmt.Printf("=== CONVERTING API RESPONSE ===\n")
	fmt.Printf("API CPU: %.2f, Memory: %.2f, Temp CPU: %.2f\n",
		newResponse.Metrics.Metrics.CPUPercent,
		newResponse.Metrics.Metrics.MemoryPercent,
		newResponse.Metrics.Metrics.Temperatures.CPU)

	// Debug log what we get from API
	s.logger.Info("DEBUG: API response before conversion",
		"cpu_percent", newResponse.Metrics.Metrics.CPUPercent,
		"memory_percent", newResponse.Metrics.Metrics.MemoryPercent,
		"temp_cpu", newResponse.Metrics.Metrics.Temperatures.CPU,
		"temp_gpu", newResponse.Metrics.Metrics.Temperatures.GPU,
		"temp_highest", newResponse.Metrics.Metrics.Temperatures.Highest)

	// Create legacy response from new API structure
	legacyResponse := &domain.LegacyMetricsResponse{
		ServerID:  newResponse.ServerID,
		ServerKey: newResponse.ServerKey,
		Metrics: domain.ServerMetrics{
			CPU:    newResponse.Metrics.Metrics.CPUPercent,
			Memory: newResponse.Metrics.Metrics.MemoryPercent,
			Disk:   newResponse.Metrics.Metrics.DiskPercent,
		},
	}

	// Convert CPU usage details
	legacyResponse.Metrics.CPUUsage = domain.CPUUsageDetails{
		UsageTotal: newResponse.Metrics.Metrics.CPUPercent,
		LoadAverage: domain.LoadAverage{
			Load1min:  newResponse.Metrics.Metrics.LoadAverage.Min1,
			Load5min:  newResponse.Metrics.Metrics.LoadAverage.Min5,
			Load15min: newResponse.Metrics.Metrics.LoadAverage.Min15,
		},
	}

	// Convert memory details
	legacyResponse.Metrics.MemoryDetails = domain.MemoryDetails{
		UsedGB:      newResponse.Metrics.Metrics.MemoryDetails.UsedGB,
		FreeGB:      newResponse.Metrics.Metrics.MemoryDetails.FreeGB,
		AvailableGB: newResponse.Metrics.Metrics.MemoryDetails.AvailableGB,
		TotalGB:     newResponse.Metrics.Metrics.MemoryDetails.UsedGB + newResponse.Metrics.Metrics.MemoryDetails.FreeGB + newResponse.Metrics.Metrics.MemoryDetails.AvailableGB,
		UsedPercent: newResponse.Metrics.Metrics.MemoryPercent,
	}

	// Convert disk details
	legacyResponse.Metrics.DiskDetails = make([]domain.DiskDetails, len(newResponse.Metrics.Metrics.DiskDetails))
	for i, disk := range newResponse.Metrics.Metrics.DiskDetails {
		legacyResponse.Metrics.DiskDetails[i] = domain.DiskDetails{
			Path:        disk.Path,
			UsedGB:      disk.UsedGB,
			FreeGB:      disk.FreeGB,
			UsedPercent: float64(disk.UsedPercent),
			TotalGB:     disk.UsedGB + disk.FreeGB,
		}
	}

	// Convert network details
	legacyResponse.Metrics.Network = newResponse.Metrics.Metrics.NetworkMbps
	legacyResponse.Metrics.NetworkDetails = domain.NetworkDetails{
		TotalRxMbps: newResponse.Metrics.Metrics.NetworkDetails.TotalRxMbps,
		TotalTxMbps: newResponse.Metrics.Metrics.NetworkDetails.TotalTxMbps,
	}

	// Convert temperature details
	legacyResponse.Metrics.TemperatureDetails = domain.TemperatureDetails{
		CPUTemperature:     newResponse.Metrics.Metrics.Temperatures.CPU,
		GPUTemperature:     newResponse.Metrics.Metrics.Temperatures.GPU,
		SystemTemperature:  newResponse.Metrics.Metrics.Temperatures.CPU, // Use CPU as system temp fallback
		HighestTemperature: newResponse.Metrics.Metrics.Temperatures.Highest,
	}

	// Log storage temperatures for debugging
	if len(newResponse.Metrics.Metrics.Temperatures.Storage) > 0 {
		s.logger.Info("Storage temperatures found", "count", len(newResponse.Metrics.Metrics.Temperatures.Storage))
		for i, storage := range newResponse.Metrics.Metrics.Temperatures.Storage {
			s.logger.Info("Storage temperature", "index", i, "device", storage.Device, "temp", storage.Temperature)
		}
	} else {
		s.logger.Info("No storage temperatures found in API response")
	}

	// Convert system details
	uptimeHours := newResponse.Metrics.Metrics.UptimeSeconds / 3600
	legacyResponse.Metrics.SystemDetails = domain.SystemDetails{
		UptimeSeconds:     newResponse.Metrics.Metrics.UptimeSeconds,
		UptimeHuman:       fmt.Sprintf("%dч %dм", uptimeHours, (newResponse.Metrics.Metrics.UptimeSeconds%3600)/60),
		ProcessesTotal:    newResponse.Metrics.Metrics.ProcessesTotal,
		ProcessesRunning:  newResponse.Metrics.Metrics.ProcessesRunning,
		ProcessesSleeping: newResponse.Metrics.Metrics.ProcessesSleeping,
	}

	return legacyResponse
}

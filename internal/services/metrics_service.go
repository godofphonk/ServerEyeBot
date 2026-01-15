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

// GetServerMetrics retrieves server metrics with caching
func (s *MetricsServiceImpl) GetServerMetrics(serverKey string) (*domain.MetricsResponse, error) {
	s.cacheMutex.RLock()

	// Check cache first
	if cached, exists := s.cache[serverKey]; exists {
		if time.Now().Before(cached.ExpiresAt) {
			s.cacheMutex.RUnlock()
			s.logger.Debug("Metrics retrieved from cache", "server_key", serverKey)
			return cached.Metrics, nil
		}
		// Cache expired, remove it
		delete(s.cache, serverKey)
	}
	s.cacheMutex.RUnlock()

	// Fetch from API
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metrics, err := s.apiClient.GetServerMetrics(ctx, serverKey)
	if err != nil {
		s.logger.Error("Failed to get server metrics", "error", err, "server_key", serverKey)
		return nil, err
	}

	// Cache the result
	s.cacheMutex.Lock()
	s.cache[serverKey] = &domain.MetricsCache{
		ServerKey: serverKey,
		Metrics:   metrics,
		ExpiresAt: time.Now().Add(60 * time.Second), // 60 seconds cache
	}
	s.cacheMutex.Unlock()

	s.logger.Info("Server metrics cached", "server_key", serverKey)
	return metrics, nil
}

// FormatCPU formats CPU metrics for display
func (s *MetricsServiceImpl) FormatCPU(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики CPU недоступны"
	}

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
	if metrics == nil || len(metrics.DiskDetails) == 0 {
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

	var sb strings.Builder
	sb.WriteString("🌡️ Температура:\n")
	sb.WriteString(fmt.Sprintf("- CPU: %.1f°C\n", metrics.TemperatureDetails.CPUTemperature))
	sb.WriteString(fmt.Sprintf("- GPU: %.1f°C\n", metrics.TemperatureDetails.GPUTemperature))
	sb.WriteString(fmt.Sprintf("- System: %.1f°C\n", metrics.TemperatureDetails.SystemTemperature))
	sb.WriteString(fmt.Sprintf("- Максимальная: %.1f°C", metrics.TemperatureDetails.HighestTemperature))

	return sb.String()
}

// FormatNetwork formats network metrics for display
func (s *MetricsServiceImpl) FormatNetwork(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики сети недоступны"
	}

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
	sb.WriteString(fmt.Sprintf("- Хостнейм: %s\n", metrics.SystemDetails.Hostname))
	sb.WriteString(fmt.Sprintf("- ОС: %s\n", metrics.SystemDetails.OS))
	sb.WriteString(fmt.Sprintf("- Ядро: %s\n", metrics.SystemDetails.Kernel))
	sb.WriteString(fmt.Sprintf("- Архитектура: %s\n", metrics.SystemDetails.Architecture))
	sb.WriteString(fmt.Sprintf("- Аптайм: %s\n", metrics.SystemDetails.UptimeHuman))
	sb.WriteString(fmt.Sprintf("- Процессы: %d (%d running)",
		metrics.SystemDetails.ProcessesTotal,
		metrics.SystemDetails.ProcessesRunning))

	return sb.String()
}

// FormatAll formats all metrics in a compact view
func (s *MetricsServiceImpl) FormatAll(metrics *domain.ServerMetrics) string {
	if metrics == nil {
		return "❌ Метрики недоступны"
	}

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

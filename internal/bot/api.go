package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereyebot/pkg/protocol"
)

// getCPUTemperature requests CPU temperature from agent via Kafka or cached metrics
func (b *Bot) getCPUTemperature(serverKey string) (float64, error) {
	// Try to get from Kafka cache first if available
	if b.useKafka && b.metricsConsumer != nil {
		temp, timestamp, err := b.metricsConsumer.GetCachedMetric(serverKey, "cpu_temperature", "")
		if err == nil {
			// Check if cached data is recent (less than 2 minutes old)
			if timestamp != nil && time.Since(*timestamp) < 2*time.Minute {
				b.logger.Debug("Using cached CPU temperature")
				return temp, nil
			}
		} else if err != nil && !strings.Contains(err.Error(), "Redis cache not available") {
			// Log unexpected errors but continue to Kafka
			b.logger.Error("Failed to get cached temperature, using Kafka", err)
		}
	}

	// Fallback to requesting from agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type tempResponse struct {
		Temperature float64 `json:"temperature"`
	}

	var temp tempResponse
	command := protocol.NewMessage(protocol.TypeGetCPUTemp, nil)

	err := b.sendCommandAndParse(ctx, serverKey, command, 30*time.Second, &temp)
	if err != nil {
		return 0, err
	}

	return temp.Temperature, nil
}

// getContainers requests Docker containers list from agent via Kafka
func (b *Bot) getContainers(serverKey string) (*protocol.ContainersPayload, error) {
	return b.getContainersViaKafka(serverKey)
}

// getContainersViaKafka requests Docker containers list from agent via Kafka
func (b *Bot) getContainersViaKafka(serverKey string) (*protocol.ContainersPayload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := protocol.NewMessage(protocol.TypeGetContainers, nil)
	response, err := b.sendCommandViaKafka(ctx, serverKey, command, 30*time.Second)
	if err != nil {
		return nil, err
	}

	result, err := protocol.ParsePayload[protocol.ContainersPayload](response)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// formatContainers formats containers list for display
func (b *Bot) formatContainers(containers *protocol.ContainersPayload) string {
	if containers.Total == 0 {
		return "📦 No Docker containers found on the server."
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("🐳 Docker Containers (%d total):\n\n", containers.Total))

	for i, container := range containers.Containers {
		if i >= 10 { // Limit to 10 containers to avoid message length issues
			result.WriteString(fmt.Sprintf("... and %d more containers\n", containers.Total-10))
			break
		}

		// Status emoji
		statusEmoji := "🔴" // Red for stopped
		if strings.Contains(strings.ToLower(container.State), "running") {
			statusEmoji = "🟢" // Green for running
		} else if strings.Contains(strings.ToLower(container.State), "paused") {
			statusEmoji = "🟡" // Yellow for paused
		}

		result.WriteString(fmt.Sprintf("%s %s\n", statusEmoji, container.Name))
		result.WriteString(fmt.Sprintf("📷 Image: `%s`\n", container.Image))
		result.WriteString(fmt.Sprintf("🔄 Status: %s\n", container.Status))

		if len(container.Ports) > 0 {
			result.WriteString(fmt.Sprintf("🔌 Ports: %s\n", strings.Join(container.Ports, ", ")))
		}

		result.WriteString("\n")
	}

	return result.String()
}

// getMemoryInfo requests memory information from agent via Streams or cached metrics
func (b *Bot) getMemoryInfo(serverKey string) (*protocol.MemoryInfo, error) {
	// Try to get from Kafka cache first if available
	if b.useKafka && b.metricsConsumer != nil {
		// Get individual memory metrics
		total, _, err1 := b.metricsConsumer.GetCachedMetric(serverKey, "memory_total", "")
		available, _, err2 := b.metricsConsumer.GetCachedMetric(serverKey, "memory_available", "")
		used, _, err3 := b.metricsConsumer.GetCachedMetric(serverKey, "memory_used", "")

		// If we have at least some cached data, use it
		if err1 == nil || err2 == nil || err3 == nil {
			b.logger.Debug("Using cached memory info")
			return &protocol.MemoryInfo{
				Total:     uint64(total),
				Available: uint64(available),
				Used:      uint64(used),
			}, nil
		}
	}

	// Fallback to requesting from agent
	var memory protocol.MemoryInfo
	return &memory, b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeGetMemoryInfo, nil), 10*time.Second, &memory)
}

// getDiskInfo requests disk information from agent via Kafka or cached metrics
func (b *Bot) getDiskInfo(serverKey string) (*protocol.DiskInfoPayload, error) {
	// Try to get from Kafka cache first if available
	if b.useKafka && b.metricsConsumer != nil {
		// Get disk usage metrics
		used, _, err1 := b.metricsConsumer.GetCachedMetric(serverKey, "disk_used", "")
		total, _, err2 := b.metricsConsumer.GetCachedMetric(serverKey, "disk_total", "")

		// If we have disk metrics, construct response
		if err1 == nil && err2 == nil && total > 0 {
			b.logger.Debug("Using cached disk info")
			percent := (used / total) * 100
			return &protocol.DiskInfoPayload{
				Disks: []protocol.DiskInfo{
					{
						Path:        "/", // Default path
						Total:       uint64(total),
						Used:        uint64(used),
						Free:        uint64(total - used),
						UsedPercent: percent,
						Filesystem:  "unknown", // Not cached
					},
				},
			}, nil
		}
	}

	// Fallback to requesting from agent
	var disk protocol.DiskInfoPayload
	return &disk, b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeGetDiskInfo, nil), 10*time.Second, &disk)
}

// getUptime requests uptime information from agent via Kafka
func (b *Bot) getUptime(serverKey string) (*protocol.UptimeInfo, error) {
	var uptime protocol.UptimeInfo
	return &uptime, b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeGetUptime, nil), 10*time.Second, &uptime)
}

// getProcesses requests processes information from agent via Kafka
func (b *Bot) getProcesses(serverKey string) (*protocol.ProcessesPayload, error) {
	var processes protocol.ProcessesPayload
	return &processes, b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeGetProcesses, nil), 10*time.Second, &processes)
}

// getNetworkInfo requests network information from agent via Kafka
func (b *Bot) getNetworkInfo(serverKey string) (*protocol.NetworkInfo, error) {
	var network protocol.NetworkInfo
	return &network, b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeGetNetworkInfo, nil), 10*time.Second, &network)
}

// updateAgent requests agent to update itself
func (b *Bot) updateAgent(serverKey string, version string, userID int64) (*protocol.UpdateAgentResponse, error) {
	payload := &protocol.UpdateAgentPayload{
		Version: version,
	}

	var update protocol.UpdateAgentResponse
	err := b.sendCommandAndParse(context.Background(), serverKey, protocol.NewMessage(protocol.TypeUpdateAgent, payload), 60*time.Second, &update)

	if err != nil {
		// Record failed update attempt
		if recordErr := b.recordAgentUpdateFailure(serverKey, version, err.Error(), userID); recordErr != nil {
			b.logger.Error("Failed to record update failure", recordErr)
		}
		return nil, err
	}

	// Record successful update
	if update.Success && update.NewVersion != "" {
		if recordErr := b.updateAgentVersion(serverKey, update.NewVersion, userID, "manual"); recordErr != nil {
			b.logger.Error("Failed to update agent version", recordErr)
		}
	}

	return &update, nil
}

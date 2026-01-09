package bot

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"time"

"github.com/servereye/servereyebot/pkg/protocol"
)

// CommandRequest represents a command request to the API
type CommandRequest struct {
ServerKey string                 `json:"server_key"`
Command   string                 `json:"command"`
Payload   map[string]interface{} `json:"payload"`
}

// CommandResponse represents a command response from the API
type CommandResponse struct {
Success bool        `json:"success"`
Data    interface{} `json:"data"`
}

// sendCommandViaHTTP sends command via HTTP API
func (b *Bot) sendCommandViaHTTP(serverKey, command string) (*CommandResponse, error) {
req := CommandRequest{
ServerKey: serverKey,
Command:   command,
}

jsonData, err := json.Marshal(req)
if err != nil {
return nil, err
}

// Use backend URL from config
url := fmt.Sprintf("%s/api/v1/commands", b.cfg.BackendURL)

resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
if err != nil {
return nil, err
}
defer resp.Body.Close()

var result CommandResponse
if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
return nil, err
}

return &result, nil
}

// getCPUTemperature requests CPU temperature from agent via HTTP API
func (b *Bot) getCPUTemperature(serverKey string) (float64, error) {
resp, err := b.sendCommandViaHTTP(serverKey, "get_cpu_temp")
if err != nil {
return 0, fmt.Errorf("Failed to get temperature: %v", err)
}

if !resp.Success {
return 0, fmt.Errorf("API returned error")
}

data, ok := resp.Data.(map[string]interface{})
if !ok {
return 0, fmt.Errorf("Invalid response format")
}

temp, ok := data["temperature"].(float64)
if !ok {
return 0, fmt.Errorf("Temperature not found in response")
}

return temp, nil
}

// getMemoryUsage requests memory usage from agent via HTTP API
func (b *Bot) getMemoryUsage(serverKey string) (float64, error) {
resp, err := b.sendCommandViaHTTP(serverKey, "get_memory_usage")
if err != nil {
return 0, fmt.Errorf("Failed to get memory usage: %v", err)
}

if !resp.Success {
return 0, fmt.Errorf("API returned error")
}

data, ok := resp.Data.(map[string]interface{})
if !ok {
return 0, fmt.Errorf("Invalid response format")
}

mem, ok := data["memory_usage"].(float64)
if !ok {
return 0, fmt.Errorf("Memory usage not found in response")
}

return mem, nil
}

// getDiskUsage requests disk usage from agent via HTTP API
func (b *Bot) getDiskUsage(serverKey string) (float64, error) {
resp, err := b.sendCommandViaHTTP(serverKey, "get_disk_usage")
if err != nil {
return 0, fmt.Errorf("Failed to get disk usage: %v", err)
}

if !resp.Success {
return 0, fmt.Errorf("API returned error")
}

data, ok := resp.Data.(map[string]interface{})
if !ok {
return 0, fmt.Errorf("Invalid response format")
}

disk, ok := data["disk_usage"].(float64)
if !ok {
return 0, fmt.Errorf("Disk usage not found in response")
}

return disk, nil
}

// getContainers requests Docker containers list from agent via HTTP API
func (b *Bot) getContainers(serverKey string) (*protocol.ContainersPayload, error) {
resp, err := b.sendCommandViaHTTP(serverKey, "get_containers")
if err != nil {
return nil, fmt.Errorf("Failed to get containers: %v", err)
}

if !resp.Success {
return nil, fmt.Errorf("API returned error")
}

// Convert response to ContainersPayload
return &protocol.ContainersPayload{
Containers: []protocol.Container{},
Timestamp:  time.Now().Unix(),
}, nil
}

package bot

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

// KeyRegistrationRequest represents a request to register a generated key
type KeyRegistrationRequest struct {
	SecretKey    string `json:"secret_key"`
	AgentVersion string `json:"agent_version,omitempty"`
	OSInfo       string `json:"os_info,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
}

// startHTTPServer starts HTTP server for agent API
func (b *Bot) startHTTPServer() {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Operation failed", nil)
		}
	}()

	b.logger.Info("Starting HTTP server for agent API")
	http.HandleFunc("/api/register-key", b.handleRegisterKey)
	http.HandleFunc("/api/validate-key/", b.handleValidateKey)
	http.HandleFunc("/api/health", b.handleHealth)
	http.HandleFunc("/api/heartbeat", b.handleHeartbeat)
	http.HandleFunc("/api/v1/servers/heartbeat", b.handleHeartbeatV1)

	http.HandleFunc("/api/monitoring/memory", b.handleMemoryRequest)
	http.HandleFunc("/api/monitoring/disk", b.handleDiskRequest)
	http.HandleFunc("/api/monitoring/uptime", b.handleUptimeRequest)
	http.HandleFunc("/api/monitoring/processes", b.handleProcessesRequest)

	// Statistics endpoints for ServerEye-Web integration
	http.HandleFunc("/api/stats/users", b.handleUserStats)

	b.logger.Info("Starting HTTP server for agent API")

	// Create HTTP server with proper timeouts for security
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		b.logger.Error("Failed to register key", err)
	}
}

// handleRegisterKey handles key registration from agent
func (b *Bot) handleRegisterKey(w http.ResponseWriter, r *http.Request) {
	b.logger.Info("Key registration request received")

	if r.Method != http.MethodPost {
		b.logger.Error("Method not allowed for key registration", nil)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req KeyRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.logger.Error("Failed to register key", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	b.logger.Info("Secret key validated")

	// Validate secret key
	if !strings.HasPrefix(req.SecretKey, "srv_") {
		http.Error(w, "Invalid secret key format", http.StatusBadRequest)
		return
	}

	// Record the key
	if err := b.recordGeneratedKey(req.SecretKey, req.Hostname); err != nil {
		b.logger.Error("Failed to register key", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// If agent info provided, update connection info
	if req.AgentVersion != "" || req.OSInfo != "" || req.Hostname != "" {
		if err := b.updateKeyConnection(req.SecretKey, req.AgentVersion, req.OSInfo, req.Hostname); err != nil {
			b.logger.Error("Failed to register key", err)
		}
	}

	b.logger.Info("Key registration completed successfully")

	b.writeJSONSuccess(w, map[string]string{
		"message": "Key registered successfully",
	})
}

// handleValidateKey handles key validation requests from ServerEye-Web
func (b *Bot) handleValidateKey(w http.ResponseWriter, r *http.Request) {
	b.logger.Info("Validate key request received", Field{"path", r.URL.Path}, Field{"method", r.Method})

	if r.Method != http.MethodGet {
		b.logger.Warn("Invalid method", Field{"method", r.Method})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract key from URL path
	path := r.URL.Path
	secretKey := strings.TrimPrefix(path, "/api/validate-key/")

	if secretKey == "" || !strings.HasPrefix(secretKey, "srv_") {
		http.Error(w, "Invalid secret key format", http.StatusBadRequest)
		return
	}

	// Check if key exists in database
	exists, err := b.keyExists(secretKey)
	if err != nil {
		b.logger.Error("Error checking key existence", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !exists {
		b.logger.Warn("Key not found", Field{"key", secretKey})
		http.Error(w, "Key not found", http.StatusNotFound)
		return
	}

	// Key exists
	b.logger.Info("Key validated successfully", Field{"key", secretKey})
	b.writeJSONSuccess(w, map[string]interface{}{
		"valid": true,
		"key":   secretKey,
	})
}

// handleHealth handles health check requests
func (b *Bot) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":  "healthy",
		"service": "servereye-bot",
	}

	// Check main database
	if b.db != nil {
		if err := b.db.Ping(); err != nil {
			health["status"] = "unhealthy"
			health["database"] = "error"
		} else {
			health["database"] = "ok"
		}
	}

	// Check keys database (non-blocking)
	if b.keysDB != nil {
		if err := b.keysDB.Ping(); err != nil {
			health["keys_database"] = "error"
			// Don't set overall status to unhealthy for keys DB failure
			b.logger.Error("Keys database health check failed", err)
		} else {
			health["keys_database"] = "ok"
		}
	}

	b.writeJSON(w, health)
}

// HeartbeatRequest represents an agent heartbeat
type HeartbeatRequest struct {
	ServerKey string `json:"server_key"`
}

// handleHeartbeat handles heartbeat requests from agents
func (b *Bot) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	b.writeJSON(w, map[string]string{
		"status":     "ok",
		"server_key": req.ServerKey,
	})
}

// handleHeartbeatV1 handles heartbeat requests from agents (v1 API with database integration)
func (b *Bot) handleHeartbeatV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		APIKey string `json:"api_key"`
		Status string `json:"status,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		b.logger.Error("Failed to decode heartbeat JSON", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate API key format
	if !strings.HasPrefix(req.APIKey, "srv_") {
		http.Error(w, "Invalid API key format", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if req.Status == "" {
		req.Status = "online"
	}

	// Update server in database
	result, err := b.db.Exec(
		"UPDATE servers SET last_seen = NOW(), status = $1, updated_at = NOW() WHERE secret_key = $2",
		req.Status, req.APIKey,
	)

	if err != nil {
		b.logger.Error("Failed to update server heartbeat", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected (valid API key)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		b.logger.Error("Failed to get rows affected", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		b.logger.Warn("Heartbeat received for unknown API key", Field{"api_key", req.APIKey})
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	b.logger.Info("Server heartbeat updated successfully", Field{"api_key", req.APIKey}, Field{"status", req.Status})

	b.writeJSON(w, map[string]string{
		"status":  "ok",
		"api_key": req.APIKey,
	})
}

// handleMemoryRequest handles direct memory requests from agents
func (b *Bot) handleMemoryRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get server key from request
	var req struct {
		ServerKey string `json:"server_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Get memory info directly
	memInfo, err := b.getMemoryInfo(req.ServerKey)
	if err != nil {
		b.logger.Error("Failed to get memory info", err)
		http.Error(w, "Failed to get memory info", http.StatusInternalServerError)
		return
	}

	// Return memory info as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    memInfo,
	})
}

// Placeholder handlers for other monitoring endpoints
func (b *Bot) handleDiskRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (b *Bot) handleUptimeRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (b *Bot) handleProcessesRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// handleUserStats returns user statistics for ServerEye-Web integration
func (b *Bot) handleUserStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	b.logger.Info("User stats request received")

	// Query total users count
	var totalUsers int64
	err := b.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		b.logger.Error("Failed to query total users", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Query active users today (updated in last 24 hours)
	var activeToday int64
	err = b.db.QueryRow(`
		SELECT COUNT(*) FROM users 
		WHERE updated_at > NOW() - INTERVAL '24 hours'
	`).Scan(&activeToday)
	if err != nil {
		b.logger.Error("Failed to query active users today", err)
		activeToday = 0 // Continue with 0 if query fails
	}

	// Query active users this week (updated in last 7 days)
	var activeWeek int64
	err = b.db.QueryRow(`
		SELECT COUNT(*) FROM users 
		WHERE updated_at > NOW() - INTERVAL '7 days'
	`).Scan(&activeWeek)
	if err != nil {
		b.logger.Error("Failed to query active users this week", err)
		activeWeek = 0 // Continue with 0 if query fails
	}

	b.logger.Info("User stats retrieved successfully")

	// Return stats in JSON format
	b.writeJSON(w, map[string]interface{}{
		"total_users":  totalUsers,
		"active_today": activeToday,
		"active_week":  activeWeek,
		"timestamp":    time.Now().Unix(),
	})
}

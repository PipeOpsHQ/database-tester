package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"tcp-stress-test/config"
	"tcp-stress-test/connectors"
	"tcp-stress-test/report"
	"tcp-stress-test/stresstester"

	"github.com/gorilla/mux"
)

// Server represents the HTTP server
type Server struct {
	config           *config.DatabaseConfig
	connectorManager *connectors.ConnectorManager
	stressTester     *stresstester.StressTester
	reportGenerator  *report.ReportGenerator
	router           *mux.Router
}

// writeJSONResponse safely writes a JSON response with proper error handling
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		// Try to send a basic error response
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"Failed to encode response","type":"encode_error"}`)
	}
}

// writeJSONError safely writes a JSON error response
func (s *Server) writeJSONError(w http.ResponseWriter, statusCode int, message string, errorType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]string{
		"error": message,
		"type":  errorType,
	}

	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		log.Printf("Error encoding JSON error response: %v", err)
		fmt.Fprintf(w, `{"error":"Failed to encode error response","type":"encode_error"}`)
	}
}

// NewServer creates a new server instance
func NewServer(cfg *config.DatabaseConfig) *Server {
	connectorManager := connectors.NewConnectorManager(cfg)
	reportGenerator := report.NewReportGenerator()

	server := &Server{
		config:           cfg,
		connectorManager: connectorManager,
		reportGenerator:  reportGenerator,
		router:           mux.NewRouter(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Static file serving for assets
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", s.healthHandler).Methods("GET")
	api.HandleFunc("/status", s.statusHandler).Methods("GET")
	api.HandleFunc("/connect", s.connectHandler).Methods("POST")
	api.HandleFunc("/disconnect", s.disconnectHandler).Methods("POST")
	api.HandleFunc("/test", s.runStressTestHandler).Methods("POST")
	api.HandleFunc("/test/{database}", s.runSingleTestHandler).Methods("POST")
	api.HandleFunc("/benchmark", s.benchmarkHandler).Methods("POST")
	api.HandleFunc("/results", s.getResultsHandler).Methods("GET")
	api.HandleFunc("/config", s.getConfigHandler).Methods("GET")
	api.HandleFunc("/config", s.updateConfigHandler).Methods("POST")
	api.HandleFunc("/config/test", s.testConfigHandler).Methods("POST")

	// Main routes
	s.router.HandleFunc("/", s.dashboardHandler).Methods("GET")
	s.router.HandleFunc("/config-ui", s.configUIHandler).Methods("GET")
	s.router.HandleFunc("/report", s.reportHandler).Methods("GET")
	s.router.HandleFunc("/live", s.liveReportHandler).Methods("GET")

	// Add middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Initialize database connections
	log.Println("Connecting to databases...")
	s.connectorManager.ConnectAll()

	// Initialize stress tester with default config
	testConfig := stresstester.DefaultTestConfig()
	allConnectors := s.connectorManager.GetAllConnectors()
	s.stressTester = stresstester.NewStressTester(allConnectors, testConfig)

	// Start server
	addr := fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Dashboard: http://%s", addr)
	log.Printf("API Health: http://%s/api/health", addr)
	log.Printf("Report: http://%s/report", addr)

	return http.ListenAndServe(addr, s.router)
}

// Stop stops the server and closes connections
func (s *Server) Stop() {
	log.Println("Shutting down server...")
	s.connectorManager.DisconnectAll()
	log.Println("Server stopped")
}

// healthHandler returns server health status
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"databases": s.stressTester.GetConnectionStatus(),
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding health response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// statusHandler returns connection status for all databases
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	connectionStatus := s.stressTester.GetConnectionStatus()
	connectivityTest := s.stressTester.RunQuickConnectivityTest()

	status := map[string]interface{}{
		"connection_status": connectionStatus,
		"connectivity_test": connectivityTest,
		"timestamp":         time.Now().Format(time.RFC3339),
		"total_databases":   len(connectionStatus),
		"connected_count":   s.countConnected(connectionStatus),
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding status response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// connectHandler attempts to connect to all databases
func (s *Server) connectHandler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in connectHandler: %v", r)
			s.writeJSONError(w, http.StatusInternalServerError, "Internal server error occurred", "panic")
		}
	}()

	log.Println("Attempting to connect to all databases...")
	s.connectorManager.ConnectAll()

	connectionStatus := s.stressTester.GetConnectionStatus()
	response := map[string]interface{}{
		"message":           "Connection attempt completed",
		"connection_status": connectionStatus,
		"connected_count":   s.countConnected(connectionStatus),
		"timestamp":         time.Now().Format(time.RFC3339),
		"success":           true,
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// disconnectHandler disconnects from all databases
func (s *Server) disconnectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Println("Disconnecting from all databases...")
	s.connectorManager.DisconnectAll()

	response := map[string]interface{}{
		"message":   "Disconnected from all databases",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding disconnect response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// runStressTestHandler runs stress tests on all connected databases
func (s *Server) runStressTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse test configuration from request
	testConfig := s.parseTestConfig(r)

	// Update stress tester with new config
	allConnectors := s.connectorManager.GetAllConnectors()
	s.stressTester = stresstester.NewStressTester(allConnectors, testConfig)

	log.Printf("Starting stress test with duration: %v, concurrency: %d",
		testConfig.Duration, testConfig.Concurrency)

	// Run the stress test
	results := s.stressTester.RunStressTest()

	response := map[string]interface{}{
		"message":     "Stress test completed",
		"results":     results,
		"config":      testConfig,
		"timestamp":   time.Now().Format(time.RFC3339),
		"total_tests": len(results),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding stress test response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// runSingleTestHandler runs stress test on a specific database
func (s *Server) runSingleTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	dbName := vars["database"]

	testConfig := s.parseTestConfig(r)

	log.Printf("Starting stress test for database: %s", dbName)

	result, err := s.stressTester.RunCustomTest(dbName, testConfig)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":    fmt.Sprintf("Failed to run test: %v", err),
			"database": dbName,
		})
		return
	}

	response := map[string]interface{}{
		"message":   fmt.Sprintf("Stress test completed for %s", dbName),
		"database":  dbName,
		"result":    result,
		"config":    testConfig,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding single test response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// benchmarkHandler runs a custom benchmark
func (s *Server) benchmarkHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var benchmarkRequest struct {
		Database   string `json:"database"`
		Query      string `json:"query"`
		Iterations int    `json:"iterations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&benchmarkRequest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if benchmarkRequest.Iterations <= 0 {
		benchmarkRequest.Iterations = 100
	}

	log.Printf("Running benchmark for %s: %d iterations",
		benchmarkRequest.Database, benchmarkRequest.Iterations)

	result, err := s.stressTester.BenchmarkQuery(
		benchmarkRequest.Database,
		benchmarkRequest.Query,
		benchmarkRequest.Iterations,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":    fmt.Sprintf("Benchmark failed: %v", err),
			"database": benchmarkRequest.Database,
		})
		return
	}

	response := map[string]interface{}{
		"message":    "Benchmark completed",
		"database":   benchmarkRequest.Database,
		"query":      benchmarkRequest.Query,
		"iterations": benchmarkRequest.Iterations,
		"result":     result,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding benchmark response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// getResultsHandler returns the latest test results
func (s *Server) getResultsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	results := s.stressTester.GetResults()
	connectionStatus := s.stressTester.GetConnectionStatus()

	response := map[string]interface{}{
		"results":           results,
		"connection_status": connectionStatus,
		"timestamp":         time.Now().Format(time.RFC3339),
		"total_results":     len(results),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding results response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// getConfigHandler returns current database configuration
func (s *Server) getConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	currentConfig := s.config.GetDatabaseConfig()

	// Mask passwords for security
	maskedConfig := s.maskPasswords(currentConfig)

	response := map[string]interface{}{
		"message":   "Current database configuration",
		"config":    maskedConfig,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding config response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to encode response"})
	}
}

// updateConfigHandler updates database configuration
func (s *Server) updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ensure we always return JSON, even in case of panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in updateConfigHandler: %v", r)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Internal server error occurred",
				"type":  "panic",
			})
		}
	}()

	var updateRequest config.DynamicDatabaseConfig

	if err := json.NewDecoder(r.Body).Decode(&updateRequest); err != nil {
		log.Printf("Error decoding request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
			"type":  "decode_error",
		}); encodeErr != nil {
			log.Printf("Failed to encode error response: %v", encodeErr)
		}
		return
	}

	// Update the global configuration
	if err := s.config.UpdateDatabaseConfig(updateRequest); err != nil {
		log.Printf("Error updating global configuration: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Failed to update global configuration: %v", err),
			"type":  "config_error",
		}); encodeErr != nil {
			log.Printf("Failed to encode error response: %v", encodeErr)
		}
		return
	}

	// Update connector manager configurations dynamically
	if err := s.connectorManager.UpdateConfigurations(updateRequest); err != nil {
		log.Printf("Warning: Some connector configurations failed to update: %v", err)
		// Don't return error here, as partial updates may still be useful
	}

	// Update stress tester with updated connectors
	allConnectors := s.connectorManager.GetAllConnectors()
	testConfig := stresstester.DefaultTestConfig()
	s.stressTester = stresstester.NewStressTester(allConnectors, testConfig)

	log.Println("Database configuration updated successfully")

	response := map[string]interface{}{
		"message":   "Database configuration updated successfully",
		"timestamp": time.Now().Format(time.RFC3339),
		"updated":   true,
		"success":   true,
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding config update response: %v", err)
		// If we can't encode the success response, try to send an error response
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to encode response",
			"type":  "encode_error",
		}); encodeErr != nil {
			log.Printf("Failed to encode error response: %v", encodeErr)
		}
	}
}

// testConfigHandler tests a database configuration before applying
func (s *Server) testConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Ensure we always return JSON, even in case of panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in testConfigHandler: %v", r)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Internal server error occurred",
				"type":  "panic",
			})
		}
	}()

	var testRequest struct {
		Database string      `json:"database"`
		Config   interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&testRequest); err != nil {
		log.Printf("Error decoding test config request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
			"type":  "decode_error",
		}); encodeErr != nil {
			log.Printf("Failed to encode error response: %v", encodeErr)
		}
		return
	}

	// Test the configuration
	err := config.TestDatabaseConnection(testRequest.Database, testRequest.Config)

	response := map[string]interface{}{
		"database":  testRequest.Database,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err != nil {
		log.Printf("Database connection test failed for %s: %v", testRequest.Database, err)
		response["success"] = false
		response["message"] = fmt.Sprintf("Connection test failed: %v", err)
		response["error"] = err.Error()
		w.WriteHeader(http.StatusOK) // Still return 200 with success: false
	} else {
		response["success"] = true
		response["message"] = "Connection test successful"
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding test config response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		if encodeErr := json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to encode response",
			"type":  "encode_error",
		}); encodeErr != nil {
			log.Printf("Failed to encode error response: %v", encodeErr)
		}
	}
}

// maskPasswords masks sensitive information in configuration
func (s *Server) maskPasswords(cfg config.DynamicDatabaseConfig) config.DynamicDatabaseConfig {
	masked := cfg

	if masked.MySQL != nil {
		maskedMySQL := *masked.MySQL
		maskedMySQL.Password = "****"
		maskedMySQL.DSN = "****"
		masked.MySQL = &maskedMySQL
	}

	if masked.PostgreSQL != nil {
		maskedPostgreSQL := *masked.PostgreSQL
		maskedPostgreSQL.Password = "****"
		maskedPostgreSQL.DSN = "****"
		masked.PostgreSQL = &maskedPostgreSQL
	}

	if masked.Redis != nil {
		maskedRedis := *masked.Redis
		if maskedRedis.Password != "" {
			maskedRedis.Password = "****"
		}
		masked.Redis = &maskedRedis
	}

	if masked.MSSQL != nil {
		maskedMSSQL := *masked.MSSQL
		maskedMSSQL.Password = "****"
		maskedMSSQL.DSN = "****"
		masked.MSSQL = &maskedMSSQL
	}

	return masked
}

// dashboardHandler serves the main dashboard
func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Database Stress Test Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .buttons { display: flex; gap: 15px; justify-content: center; margin: 30px 0; flex-wrap: wrap; }
        .btn { padding: 12px 24px; border: none; border-radius: 5px; cursor: pointer; text-decoration: none; font-size: 16px; display: inline-block; }
        .btn-primary { background: #007bff; color: white; }
        .btn-success { background: #28a745; color: white; }
        .btn-warning { background: #ffc107; color: black; }
        .btn-info { background: #17a2b8; color: white; }
        .btn-secondary { background: #6c757d; color: white; }
        .btn:hover { opacity: 0.8; transform: translateY(-1px); transition: all 0.2s; }
        .status { margin: 20px 0; padding: 15px; background: #f8f9fa; border-radius: 5px; }
        .endpoints { margin: 30px 0; }
        .endpoint { margin: 10px 0; padding: 10px; background: #e9ecef; border-radius: 4px; font-family: monospace; }
        .footer { text-align: center; margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666; }
        .config-section { margin: 20px 0; padding: 15px; background: #fff3cd; border-radius: 5px; border-left: 4px solid #ffc107; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Database Stress Test Dashboard</h1>
        <p style="text-align: center; color: #666; margin-bottom: 30px;">
            PipeOps TCP/UDP Port Testing Service - Multi-Database Stress Testing Platform
        </p>

        <div class="buttons">
            <a href="/report" class="btn btn-primary">View Full Report</a>
            <a href="/live" class="btn btn-success">Live Report</a>
            <a href="/config-ui" class="btn btn-secondary">Configure Databases</a>
            <a href="/api/status" class="btn btn-info">Connection Status</a>
            <a href="/api/health" class="btn btn-warning">Health Check</a>
        </div>

        <div class="config-section">
            <h3>Database Configuration</h3>
            <p>Use the <strong>Configure Databases</strong> button above to easily add and test your database credentials through the web interface.</p>
            <div class="buttons">
                <button onclick="openConfigUI()" class="btn btn-secondary">Open Configuration UI</button>
                <button onclick="viewCurrentConfig()" class="btn btn-info">View Current Config</button>
            </div>
        </div>

        <div class="status">
            <h3>Quick Actions</h3>
            <div class="buttons">
                <button onclick="connectAll()" class="btn btn-success">Connect All DBs</button>
                <button onclick="runStressTest()" class="btn btn-primary">Run Stress Test</button>
                <button onclick="checkStatus()" class="btn btn-info">Check Status</button>
                <button onclick="disconnectAll()" class="btn btn-warning">Disconnect All</button>
            </div>
        </div>

        <div class="endpoints">
            <h3>Available Endpoints</h3>
            <div class="endpoint">GET  / - Dashboard (this page)</div>
            <div class="endpoint">GET  /config-ui - Database configuration interface</div>
            <div class="endpoint">GET  /report - Full HTML report</div>
            <div class="endpoint">GET  /live - Live updating report</div>
            <div class="endpoint">GET  /api/health - Health check</div>
            <div class="endpoint">GET  /api/status - Database connection status</div>
            <div class="endpoint">GET  /api/config - Get current database configuration</div>
            <div class="endpoint">POST /api/config - Update database configuration</div>
            <div class="endpoint">POST /api/config/test - Test database connection</div>
            <div class="endpoint">POST /api/connect - Connect to all databases</div>
            <div class="endpoint">POST /api/disconnect - Disconnect from all databases</div>
            <div class="endpoint">POST /api/test - Run stress test on all databases</div>
            <div class="endpoint">POST /api/test/{database} - Run stress test on specific database</div>
            <div class="endpoint">POST /api/benchmark - Run custom benchmark</div>
            <div class="endpoint">GET  /api/results - Get latest test results</div>
        </div>

        <div class="footer">
            <p>Database Stress Test Service v1.0.0</p>
            <p>Supports: MySQL, PostgreSQL, MongoDB, Redis, Microsoft SQL Server</p>
            <p>Built for PipeOps TCP/UDP port testing</p>
        </div>
    </div>

    <script>
        async function connectAll() {
            try {
                const response = await fetch('/api/connect', { method: 'POST' });
                const data = await response.json();
                alert('Connect attempt completed. Check status for details.');
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        async function runStressTest() {
            if (!confirm('Run stress test on all connected databases? This may take some time.')) return;

            try {
                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 10, query_type: 'simple' })
                });
                const data = await response.json();
                alert('Stress test completed! Check the report page for details.');
                window.open('/report', '_blank');
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        async function checkStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                let message = 'Database Status:\n';
                for (const [db, status] of Object.entries(data.connection_status)) {
                    message += db + ': ' + status + '\n';
                }
                alert(message);
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        async function disconnectAll() {
            if (!confirm('Disconnect from all databases?')) return;

            try {
                const response = await fetch('/api/disconnect', { method: 'POST' });
                const data = await response.json();
                alert('Disconnected from all databases.');
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        function openConfigUI() {
            window.open('/config-ui', '_blank');
        }

        async function viewCurrentConfig() {
            try {
                const response = await fetch('/api/config');
                const data = await response.json();
                console.log('Current configuration:', data);
                alert('Current configuration loaded. Check browser console for details.');
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// configUIHandler serves the database configuration interface
func (s *Server) configUIHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Database Configuration - Stress Test Service</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #f5f5f5; line-height: 1.6; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 10px; margin-bottom: 30px; text-align: center; }
        .header h1 { font-size: 2.5em; margin-bottom: 10px; }
        .header .subtitle { font-size: 1.2em; opacity: 0.9; }

        .nav-buttons { display: flex; gap: 15px; justify-content: center; margin-bottom: 30px; flex-wrap: wrap; }
        .btn { padding: 12px 24px; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; font-weight: 600; text-decoration: none; display: inline-block; transition: all 0.3s; }
        .btn-primary { background: #007bff; color: white; }
        .btn-success { background: #28a745; color: white; }
        .btn-warning { background: #ffc107; color: #212529; }
        .btn-danger { background: #dc3545; color: white; }
        .btn-secondary { background: #6c757d; color: white; }
        .btn:hover { transform: translateY(-2px); box-shadow: 0 4px 8px rgba(0,0,0,0.2); }

        .database-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 25px; margin-bottom: 30px; }
        .database-card { background: white; border-radius: 15px; padding: 25px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); border-left: 5px solid #007bff; }
        .database-card.mysql { border-left-color: #f39c12; }
        .database-card.postgresql { border-left-color: #336791; }
        .database-card.redis { border-left-color: #d82c20; }
        .database-card.mssql { border-left-color: #cc2927; }
        .database-card.mongodb { border-left-color: #4db33d; }

        .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
        .card-title { font-size: 1.4em; font-weight: bold; color: #333; }
        .status-indicator { padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: bold; }
        .status-connected { background: #d4edda; color: #155724; }
        .status-disconnected { background: #f8d7da; color: #721c24; }
        .status-unknown { background: #fff3cd; color: #856404; }

        .form-group { margin-bottom: 15px; }
        .form-label { display: block; margin-bottom: 5px; font-weight: 600; color: #555; }
        .form-input { width: 100%; padding: 10px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; transition: border-color 0.3s; }
        .form-input:focus { outline: none; border-color: #007bff; }
        .form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 15px; }

        .card-actions { display: flex; gap: 10px; margin-top: 20px; flex-wrap: wrap; }
        .btn-small { padding: 8px 16px; font-size: 12px; }

        .global-actions { background: white; padding: 25px; border-radius: 15px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); text-align: center; }

        .message { padding: 15px; border-radius: 8px; margin: 15px 0; display: none; }
        .message.success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .message.error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .message.info { background: #cce7ff; color: #004085; border: 1px solid #b8daff; }

        .loading { display: none; text-align: center; padding: 20px; }
        .spinner { display: inline-block; width: 20px; height: 20px; border: 3px solid #f3f3f3; border-top: 3px solid #007bff; border-radius: 50%; animation: spin 1s linear infinite; }
        @keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }

        .collapsible { cursor: pointer; user-select: none; }
        .collapsible:after { content: '▼'; float: right; margin-left: 10px; transition: transform 0.3s; }
        .collapsible.collapsed:after { transform: rotate(-90deg); }
        .collapsible-content { display: block; overflow: hidden; transition: max-height 0.3s ease-out; }
        .collapsible-content.collapsed { max-height: 0; }

        .modal {
            position: fixed; top: 0; left: 0; width: 100%; height: 100%;
            background: rgba(0,0,0,0.5); z-index: 1000;
            display: flex; align-items: center; justify-content: center;
        }
        .modal-content {
            background: white; padding: 30px; border-radius: 10px; width: 90%; max-width: 600px;
            max-height: 90vh; overflow-y: auto;
        }
        .modal-actions {
            margin-top: 20px; text-align: right;
        }
        .modal-actions .btn { margin-left: 10px; }
        .form-section { margin: 20px 0; padding: 15px; border: 1px solid #e0e0e0; border-radius: 5px; }
        .form-section h4 { margin-bottom: 15px; color: #333; }

        @media (max-width: 768px) {
            .database-grid { grid-template-columns: 1fr; }
            .form-row { grid-template-columns: 1fr; }
            .nav-buttons { flex-direction: column; align-items: center; }
            .modal-content { width: 95%; padding: 20px; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Database Configuration</h1>
            <div class="subtitle">Configure and test your database connections</div>
        </div>

        <div class="nav-buttons">
            <a href="/" class="btn btn-secondary">Dashboard</a>
            <button onclick="loadCurrentConfig()" class="btn btn-primary">Reload Config</button>
            <button onclick="saveAllConfigs()" class="btn btn-success">Save All</button>
            <button onclick="testAllConnections()" class="btn btn-warning">Test All</button>
        </div>

        <div id="message-container"></div>
        <div id="loading" class="loading">
            <div class="spinner"></div>
            <span>Processing...</span>
        </div>

        <div class="database-grid">
            <!-- MySQL Configuration -->
            <div class="database-card mysql">
                <div class="card-header">
                    <div class="card-title collapsible" onclick="toggleCard('mysql')">
                        MySQL Database
                    </div>
                    <div class="status-indicator status-unknown" id="mysql-status">Unknown</div>
                </div>
                <div class="collapsible-content" id="mysql-content">
                    <form id="mysql-form">
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Host</label>
                                <input type="text" class="form-input" name="host" placeholder="localhost" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Port</label>
                                <input type="number" class="form-input" name="port" placeholder="3306" required>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Username</label>
                                <input type="text" class="form-input" name="username" placeholder="root" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Password</label>
                                <input type="password" class="form-input" name="password" placeholder="Enter password" required>
                            </div>
                        </div>
                        <div class="form-group">
                            <label class="form-label">Database Name</label>
                            <input type="text" class="form-input" name="database" placeholder="testdb" required>
                        </div>
                    </form>
                    <div class="card-actions">
                        <button onclick="testConnection('mysql')" class="btn btn-warning btn-small">Test Connection</button>
                        <button onclick="saveConfig('mysql')" class="btn btn-success btn-small">Save Config</button>
                    </div>
                </div>
            </div>

            <!-- PostgreSQL Configuration -->
            <div class="database-card postgresql">
                <div class="card-header">
                    <div class="card-title collapsible" onclick="toggleCard('postgresql')">
                        PostgreSQL Database
                    </div>
                    <div class="status-indicator status-unknown" id="postgresql-status">Unknown</div>
                </div>
                <div class="collapsible-content" id="postgresql-content">
                    <form id="postgresql-form">
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Host</label>
                                <input type="text" class="form-input" name="host" placeholder="localhost" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Port</label>
                                <input type="number" class="form-input" name="port" placeholder="5432" required>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Username</label>
                                <input type="text" class="form-input" name="username" placeholder="postgres" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Password</label>
                                <input type="password" class="form-input" name="password" placeholder="Enter password" required>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Database Name</label>
                                <input type="text" class="form-input" name="database" placeholder="testdb" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">SSL Mode</label>
                                <select class="form-input" name="sslmode">
                                    <option value="disable">Disable</option>
                                    <option value="require">Require</option>
                                    <option value="prefer">Prefer</option>
                                </select>
                            </div>
                        </div>
                    </form>
                    <div class="card-actions">
                        <button onclick="testConnection('postgresql')" class="btn btn-warning btn-small">Test Connection</button>
                        <button onclick="saveConfig('postgresql')" class="btn btn-success btn-small">Save Config</button>
                    </div>
                </div>
            </div>

            <!-- Redis Configuration -->
            <div class="database-card redis">
                <div class="card-header">
                    <div class="card-title collapsible" onclick="toggleCard('redis')">
                        Redis Cache
                    </div>
                    <div class="status-indicator status-unknown" id="redis-status">Unknown</div>
                </div>
                <div class="collapsible-content" id="redis-content">
                    <form id="redis-form">
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Host</label>
                                <input type="text" class="form-input" name="host" placeholder="localhost" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Port</label>
                                <input type="number" class="form-input" name="port" placeholder="6379" required>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Password</label>
                                <input type="password" class="form-input" name="password" placeholder="Leave empty if none">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Database Number</label>
                                <input type="number" class="form-input" name="db" placeholder="0" min="0" max="15">
                            </div>
                        </div>
                    </form>
                    <div class="card-actions">
                        <button onclick="testConnection('redis')" class="btn btn-warning btn-small">Test Connection</button>
                        <button onclick="saveConfig('redis')" class="btn btn-success btn-small">Save Config</button>
                    </div>
                </div>
            </div>

            <!-- MSSQL Configuration -->
            <div class="database-card mssql">
                <div class="card-header">
                    <div class="card-title collapsible" onclick="toggleCard('mssql')">
                        Microsoft SQL Server
                    </div>
                    <div class="status-indicator status-unknown" id="mssql-status">Unknown</div>
                </div>
                <div class="collapsible-content" id="mssql-content">
                    <form id="mssql-form">
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Host</label>
                                <input type="text" class="form-input" name="host" placeholder="localhost" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Port</label>
                                <input type="number" class="form-input" name="port" placeholder="1433" required>
                            </div>
                        </div>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Username</label>
                                <input type="text" class="form-input" name="username" placeholder="sa" required>
                            </div>
                            <div class="form-group">
                                <label class="form-label">Password</label>
                                <input type="password" class="form-input" name="password" placeholder="Enter password" required>
                            </div>
                        </div>
                        <div class="form-group">
                            <label class="form-label">Database Name</label>
                            <input type="text" class="form-input" name="database" placeholder="testdb" required>
                        </div>
                    </form>
                    <div class="card-actions">
                        <button onclick="testConnection('mssql')" class="btn btn-warning btn-small">Test Connection</button>
                        <button onclick="saveConfig('mssql')" class="btn btn-success btn-small">Save Config</button>
                    </div>
                </div>
            </div>

            <!-- MongoDB Configuration -->
            <div class="database-card mongodb">
                <div class="card-header">
                    <div class="card-title collapsible" onclick="toggleCard('mongodb')">
                        MongoDB Database
                    </div>
                    <div class="status-indicator status-unknown" id="mongodb-status">Unknown</div>
                </div>
                <div class="collapsible-content" id="mongodb-content">
                    <form id="mongodb-form">
                        <div class="form-group">
                            <label class="form-label">Connection URI</label>
                            <input type="text" class="form-input" name="uri" placeholder="mongodb://localhost:27017" required>
                        </div>
                        <div class="form-group">
                            <label class="form-label">Database Name</label>
                            <input type="text" class="form-input" name="database" placeholder="testdb" required>
                        </div>
                    </form>
                    <div class="card-actions">
                        <button onclick="testConnection('mongodb')" class="btn btn-warning btn-small">Test Connection</button>
                        <button onclick="saveConfig('mongodb')" class="btn btn-success btn-small">Save Config</button>
                    </div>
                </div>
            </div>
        </div>

        <div class="global-actions">
            <h3>Quick Actions</h3>
            <div class="nav-buttons" style="margin-top: 15px;">
                <button onclick="connectAllDatabases()" class="btn btn-success">Connect All</button>
                <button onclick="showAdvancedTestOptions()" class="btn btn-primary">Advanced Stress Test</button>
                <button onclick="runQuickStressTest()" class="btn btn-warning">Quick Test</button>
                <button onclick="viewReport()" class="btn btn-info">View Report</button>
            </div>
        </div>

        <!-- Advanced Test Configuration Modal -->
        <div id="advanced-test-modal" class="modal" style="display: none;">
            <div class="modal-content">
                <h3>Advanced Stress Test Configuration</h3>
                <form id="advanced-test-form">
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Test Duration (seconds)</label>
                            <input type="number" class="form-input" name="duration" value="30" min="5" max="3600">
                        </div>
                        <div class="form-group">
                            <label class="form-label">Concurrency Level</label>
                            <input type="number" class="form-input" name="concurrency" value="10" min="1" max="100">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">Query Type</label>
                            <select class="form-input" name="query_type" onchange="toggleMixedRatioOptions()">
                                <option value="simple">Simple</option>
                                <option value="complex">Complex</option>
                                <option value="heavy_read">Heavy Read</option>
                                <option value="heavy_write">Heavy Write</option>
                                <option value="mixed">Mixed Workload</option>
                                <option value="transaction">Transaction</option>
                                <option value="bulk">Bulk Operations</option>
                                <option value="analytics">Analytics</option>
                                <option value="concurrent">Concurrent</option>
                            </select>
                        </div>
                        <div class="form-group">
                            <label class="form-label">Delay Between Operations (ms)</label>
                            <input type="number" class="form-input" name="delay_between" value="100" min="0" max="5000">
                        </div>
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label class="form-label">
                                <input type="checkbox" name="burst_mode"> Burst Mode (No delays)
                            </label>
                        </div>
                        <div class="form-group">
                            <label class="form-label">
                                <input type="checkbox" name="randomize_queries" checked> Randomize Queries
                            </label>
                        </div>
                    </div>

                    <!-- Mixed Workload Ratios -->
                    <div id="mixed-ratio-options" class="form-section" style="display: none;">
                        <h4>Mixed Workload Ratios</h4>
                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Read Operations (%)</label>
                                <input type="number" class="form-input" name="read_percent" value="70" min="0" max="100">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Write Operations (%)</label>
                                <input type="number" class="form-input" name="write_percent" value="25" min="0" max="100">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Transaction Operations (%)</label>
                                <input type="number" class="form-input" name="transaction_percent" value="5" min="0" max="100">
                            </div>
                        </div>
                    </div>

                    <!-- Preset Configurations -->
                    <div class="form-section">
                        <h4>Preset Configurations</h4>
                        <div class="nav-buttons">
                            <button type="button" onclick="applyPreset('light')" class="btn btn-secondary btn-small">Light Load</button>
                            <button type="button" onclick="applyPreset('moderate')" class="btn btn-secondary btn-small">Moderate Load</button>
                            <button type="button" onclick="applyPreset('heavy')" class="btn btn-secondary btn-small">Heavy Load</button>
                            <button type="button" onclick="applyPreset('extreme')" class="btn btn-secondary btn-small">Extreme Load</button>
                        </div>
                    </div>
                </form>

                <div class="modal-actions">
                    <button onclick="runAdvancedStressTest()" class="btn btn-primary">Run Test</button>
                    <button onclick="hideAdvancedTestOptions()" class="btn btn-secondary">Cancel</button>
                </div>
            </div>
        </div>
    </div>

    <script>
        let currentConfig = {};

        // Load current configuration on page load
        document.addEventListener('DOMContentLoaded', function() {
            loadCurrentConfig();
        });

        function showMessage(text, type = 'info') {
            const container = document.getElementById('message-container');
            const message = document.createElement('div');
            message.className = 'message ' + type;
            message.textContent = text;
            message.style.display = 'block';

            container.innerHTML = '';
            container.appendChild(message);

            setTimeout(() => {
                message.style.display = 'none';
            }, 5000);
        }

        function showLoading(show = true) {
            document.getElementById('loading').style.display = show ? 'block' : 'none';
        }

        function toggleCard(database) {
            const content = document.getElementById(database + '-content');
            const header = content.previousElementSibling.querySelector('.collapsible');

            content.classList.toggle('collapsed');
            header.classList.toggle('collapsed');
        }

        async function loadCurrentConfig() {
            try {
                showLoading(true);
                const response = await fetch('/api/config');
                const data = await response.json();

                if (data.config) {
                    currentConfig = data.config;
                    populateForms(currentConfig);
                    await updateConnectionStatuses();
                    showMessage('Configuration loaded successfully', 'success');
                } else {
                    showMessage('Failed to load configuration', 'error');
                }
            } catch (error) {
                showMessage('Error loading configuration: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        function populateForms(config) {
            // Populate MySQL form
            if (config.mysql) {
                const form = document.getElementById('mysql-form');
                form.host.value = config.mysql.host || '';
                form.port.value = config.mysql.port || '';
                form.username.value = config.mysql.username || '';
                form.database.value = config.mysql.database || '';
                // Don't populate password for security
            }

            // Populate PostgreSQL form
            if (config.postgresql) {
                const form = document.getElementById('postgresql-form');
                form.host.value = config.postgresql.host || '';
                form.port.value = config.postgresql.port || '';
                form.username.value = config.postgresql.username || '';
                form.database.value = config.postgresql.database || '';
                form.sslmode.value = config.postgresql.sslmode || 'disable';
                // Don't populate password for security
            }

            // Populate Redis form
            if (config.redis) {
                const form = document.getElementById('redis-form');
                form.host.value = config.redis.host || '';
                form.port.value = config.redis.port || '';
                form.db.value = config.redis.db || '';
                // Don't populate password for security
            }

            // Populate MSSQL form
            if (config.mssql) {
                const form = document.getElementById('mssql-form');
                form.host.value = config.mssql.host || '';
                form.port.value = config.mssql.port || '';
                form.username.value = config.mssql.username || '';
                form.database.value = config.mssql.database || '';
                // Don't populate password for security
            }

            // Populate MongoDB form
            if (config.mongodb) {
                const form = document.getElementById('mongodb-form');
                form.uri.value = config.mongodb.uri || '';
                form.database.value = config.mongodb.database || '';
            }
        }

        async function updateConnectionStatuses() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();

                if (data.connection_status) {
                    Object.keys(data.connection_status).forEach(db => {
                        const statusEl = document.getElementById(db.toLowerCase() + '-status');
                        if (statusEl) {
                            const status = data.connection_status[db];
                            statusEl.textContent = status;
                            statusEl.className = 'status-indicator ' +
                                (status === 'Connected' ? 'status-connected' : 'status-disconnected');
                        }
                    });
                }
            } catch (error) {
                console.error('Failed to update connection statuses:', error);
            }
        }

        function getFormData(database) {
            const form = document.getElementById(database + '-form');
            const formData = new FormData(form);
            const data = {};

            for (let [key, value] of formData.entries()) {
                if (key === 'db' && database === 'redis') {
                    // Redis DB should be an integer
                    data[key] = parseInt(value) || 0;
                } else if (key === 'port') {
                    // All ports should be strings for consistency with Go structs
                    data[key] = value.toString();
                } else {
                    data[key] = value;
                }
            }

            return data;
        }

        async function testConnection(database) {
            try {
                showLoading(true);
                const formData = getFormData(database);

                const response = await fetch('/api/config/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        database: database,
                        config: formData
                    })
                });

                let data;
                const contentType = response.headers.get('content-type');

                if (contentType && contentType.includes('application/json')) {
                    try {
                        data = await response.json();
                    } catch (jsonError) {
                        throw new Error('Invalid JSON response from server');
                    }
                } else {
                    const text = await response.text();
                    throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                }

                if (data.success) {
                    showMessage(database.toUpperCase() + ' connection test successful!', 'success');
                    updateStatus(database, 'Connected');
                } else {
                    showMessage(database.toUpperCase() + ' connection failed: ' + data.message, 'error');
                    updateStatus(database, 'Disconnected');
                }
            } catch (error) {
                showMessage('Connection test error: ' + error.message, 'error');
                updateStatus(database, 'Disconnected');
            } finally {
                showLoading(false);
            }
        }

        function updateStatus(database, status) {
            const statusEl = document.getElementById(database + '-status');
            if (statusEl) {
                statusEl.textContent = status;
                statusEl.className = 'status-indicator ' +
                    (status === 'Connected' ? 'status-connected' : 'status-disconnected');
            }
        }

        async function saveConfig(database) {
            try {
                showLoading(true);
                const formData = getFormData(database);
                const updateData = {};
                updateData[database] = formData;

                const response = await fetch('/api/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(updateData)
                });

                let data;
                const contentType = response.headers.get('content-type');

                if (contentType && contentType.includes('application/json')) {
                    try {
                        data = await response.json();
                    } catch (jsonError) {
                        throw new Error('Invalid JSON response from server');
                    }
                } else {
                    const text = await response.text();
                    throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                }

                if (response.ok) {
                    showMessage(database.toUpperCase() + ' configuration saved successfully!', 'success');
                    await updateConnectionStatuses();
                } else {
                    const errorMsg = data.error || data.message || 'Unknown error occurred';
                    showMessage('Failed to save ' + database + ' configuration: ' + errorMsg, 'error');
                }
            } catch (error) {
                showMessage('Save error: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        async function saveAllConfigs() {
            const databases = ['mysql', 'postgresql', 'redis', 'mssql', 'mongodb'];
            const updateData = {};

            databases.forEach(db => {
                updateData[db] = getFormData(db);
            });

            try {
                showLoading(true);
                const response = await fetch('/api/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(updateData)
                });

                if (response.ok) {
                    showMessage('All database configurations saved successfully!', 'success');
                    await updateConnectionStatuses();
                } else {
                    let data;
                    const contentType = response.headers.get('content-type');

                    if (contentType && contentType.includes('application/json')) {
                        try {
                            data = await response.json();
                        } catch (jsonError) {
                            throw new Error('Invalid JSON response from server');
                        }
                    } else {
                        const text = await response.text();
                        throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                    }

                    const errorMsg = data.error || data.message || 'Unknown error occurred';
                    showMessage('Failed to save configurations: ' + errorMsg, 'error');
                }
            } catch (error) {
                showMessage('Save error: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        async function testAllConnections() {
            const databases = ['mysql', 'postgresql', 'redis', 'mssql', 'mongodb'];
            let results = [];

            showLoading(true);

            for (const db of databases) {
                try {
                    const formData = getFormData(db);
                    const response = await fetch('/api/config/test', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                            database: db,
                            config: formData
                        })
                    });

                    let data;
                    const contentType = response.headers.get('content-type');

                    if (contentType && contentType.includes('application/json')) {
                        try {
                            data = await response.json();
                        } catch (jsonError) {
                            throw new Error('Invalid JSON response from server');
                        }
                    } else {
                        const text = await response.text();
                        throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                    }
                    results.push({
                        database: db,
                        success: data.success,
                        message: data.message
                    });

                    updateStatus(db, data.success ? 'Connected' : 'Disconnected');
                } catch (error) {
                    results.push({
                        database: db,
                        success: false,
                        message: error.message
                    });
                    updateStatus(db, 'Disconnected');
                }
            }

            showLoading(false);

            const successCount = results.filter(r => r.success).length;
            const totalCount = results.length;

            if (successCount === totalCount) {
                showMessage('All connection tests passed! (' + successCount + '/' + totalCount + ')', 'success');
            } else {
                showMessage('Connection tests completed: ' + successCount + '/' + totalCount + ' passed', 'error');
            }
        }

        async function connectAllDatabases() {
            try {
                showLoading(true);
                const response = await fetch('/api/connect', { method: 'POST' });

                let data;
                const contentType = response.headers.get('content-type');

                if (contentType && contentType.includes('application/json')) {
                    try {
                        data = await response.json();
                    } catch (jsonError) {
                        throw new Error('Invalid JSON response from server');
                    }
                } else {
                    const text = await response.text();
                    throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                }

                showMessage('Database connection attempt completed. Check individual statuses.', 'info');
                await updateConnectionStatuses();
            } catch (error) {
                showMessage('Connection error: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        async function runQuickStressTest() {
            if (!confirm('Run a quick stress test on all connected databases? This will take about 30 seconds.')) {
                return;
            }

            try {
                showLoading(true);
                showMessage('Starting quick stress test...', 'info');

                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        duration: 30,
                        concurrency: 10,
                        query_type: 'simple'
                    })
                });

                let data;
                const contentType = response.headers.get('content-type');

                if (contentType && contentType.includes('application/json')) {
                    try {
                        data = await response.json();
                    } catch (jsonError) {
                        throw new Error('Invalid JSON response from server');
                    }
                } else {
                    const text = await response.text();
                    throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                }
                showMessage('Quick stress test completed! View the report to see results.', 'success');
            } catch (error) {
                showMessage('Stress test error: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        function showAdvancedTestOptions() {
            document.getElementById('advanced-test-modal').style.display = 'block';
        }

        function hideAdvancedTestOptions() {
            document.getElementById('advanced-test-modal').style.display = 'none';
        }

        function toggleMixedRatioOptions() {
            const queryType = document.querySelector('select[name="query_type"]').value;
            const mixedOptions = document.getElementById('mixed-ratio-options');
            mixedOptions.style.display = queryType === 'mixed' ? 'block' : 'none';
        }

        function applyPreset(preset) {
            const form = document.getElementById('advanced-test-form');

            switch(preset) {
                case 'light':
                    form.duration.value = 30;
                    form.concurrency.value = 5;
                    form.query_type.value = 'simple';
                    form.delay_between.value = 200;
                    form.burst_mode.checked = false;
                    break;
                case 'moderate':
                    form.duration.value = 60;
                    form.concurrency.value = 15;
                    form.query_type.value = 'mixed';
                    form.delay_between.value = 100;
                    form.burst_mode.checked = false;
                    form.read_percent.value = 70;
                    form.write_percent.value = 25;
                    form.transaction_percent.value = 5;
                    break;
                case 'heavy':
                    form.duration.value = 120;
                    form.concurrency.value = 30;
                    form.query_type.value = 'heavy_read';
                    form.delay_between.value = 50;
                    form.burst_mode.checked = true;
                    break;
                case 'extreme':
                    form.duration.value = 300;
                    form.concurrency.value = 50;
                    form.query_type.value = 'mixed';
                    form.delay_between.value = 10;
                    form.burst_mode.checked = true;
                    form.read_percent.value = 60;
                    form.write_percent.value = 30;
                    form.transaction_percent.value = 10;
                    break;
            }
            toggleMixedRatioOptions();
        }

        async function runAdvancedStressTest() {
            const form = document.getElementById('advanced-test-form');
            const formData = new FormData(form);

            const config = {
                duration: parseInt(formData.get('duration')),
                concurrency: parseInt(formData.get('concurrency')),
                query_type: formData.get('query_type'),
                delay_between: parseInt(formData.get('delay_between')),
                burst_mode: formData.get('burst_mode') === 'on',
                randomize_queries: formData.get('randomize_queries') === 'on'
            };

            // Add mixed workload ratios if applicable
            if (config.query_type === 'mixed') {
                config.read_percent = parseInt(formData.get('read_percent'));
                config.write_percent = parseInt(formData.get('write_percent'));
                config.transaction_percent = parseInt(formData.get('transaction_percent'));

                // Validate percentages sum to 100
                const total = config.read_percent + config.write_percent + config.transaction_percent;
                if (total !== 100) {
                    showMessage('Mixed workload percentages must sum to 100%', 'error');
                    return;
                }
            }

            if (!confirm('Run advanced stress test with ' + config.concurrency + ' concurrent operations for ' + config.duration + ' seconds using ' + config.query_type + ' queries?')) {
                return;
            }

            try {
                hideAdvancedTestOptions();
                showLoading(true);
                showMessage('Starting advanced stress test (' + config.query_type + ')... This may take ' + config.duration + ' seconds or more.', 'info');

                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(config)
                });

                let data;
                const contentType = response.headers.get('content-type');

                if (contentType && contentType.includes('application/json')) {
                    try {
                        data = await response.json();
                    } catch (jsonError) {
                        throw new Error('Invalid JSON response from server');
                    }
                } else {
                    const text = await response.text();
                    throw new Error('Server returned non-JSON response: ' + text.substring(0, 100));
                }
                showMessage('Advanced stress test completed! View the report to see detailed results.', 'success');
            } catch (error) {
                showMessage('Advanced stress test error: ' + error.message, 'error');
            } finally {
                showLoading(false);
            }
        }

        function viewReport() {
            window.open('/report', '_blank');
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// reportHandler serves the HTML report
func (s *Server) reportHandler(w http.ResponseWriter, r *http.Request) {
	results := s.stressTester.GetResults()
	connectionStatus := s.stressTester.GetConnectionStatus()

	if len(results) == 0 {
		// No test results yet, show a message
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>No Test Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; text-align: center; background: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .btn { padding: 12px 24px; border: none; border-radius: 5px; cursor: pointer; text-decoration: none; font-size: 16px; background: #007bff; color: white; margin: 10px; }
        .btn:hover { opacity: 0.8; }
    </style>
</head>
<body>
    <div class="container">
        <h1>No Test Results Available</h1>
        <p>No stress tests have been run yet. Run a test to see the report.</p>
        <a href="/" class="btn">Go to Dashboard</a>
        <button onclick="runTest()" class="btn">Run Test Now</button>
    </div>
    <script>
        async function runTest() {
            if (!confirm('Run stress test now? This will take about 30 seconds.')) return;
            try {
                await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 5 })
                });
                window.location.reload();
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}

	htmlReport, err := s.reportGenerator.GenerateHTML(results, connectionStatus)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate report: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlReport))
}

// liveReportHandler serves a live-updating report page
func (s *Server) liveReportHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Live Database Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #007bff; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; text-align: center; }
        .controls { background: white; padding: 15px; border-radius: 8px; margin-bottom: 20px; text-align: center; }
        .btn { padding: 8px 16px; margin: 0 5px; border: none; border-radius: 4px; cursor: pointer; }
        .btn-primary { background: #007bff; color: white; }
        .btn-success { background: #28a745; color: white; }
        .status { margin: 10px 0; padding: 10px; border-radius: 4px; }
        .status.loading { background: #fff3cd; }
        .status.success { background: #d4edda; }
        .status.error { background: #f8d7da; }
        #report-container { background: white; border-radius: 8px; overflow: hidden; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Live Database Stress Test Report</h1>
            <div id="last-updated">Loading...</div>
        </div>

        <div class="controls">
            <button onclick="refreshNow()" class="btn btn-primary">Refresh Now</button>
            <button onclick="runNewTest()" class="btn btn-success">Run New Test</button>
            <span id="auto-refresh-status">Auto-refresh: ON</span>
            <button onclick="toggleAutoRefresh()" class="btn btn-secondary" id="toggle-btn">Pause</button>
        </div>

        <div id="status" class="status">Ready</div>
        <div id="report-container">Loading report...</div>
    </div>

    <script>
        let autoRefresh = true;
        let refreshInterval;

        function updateStatus(message, type = '') {
            const status = document.getElementById('status');
            status.textContent = message;
            status.className = 'status ' + type;
        }

        async function loadReport() {
            try {
                updateStatus('Loading latest report...', 'loading');
                const response = await fetch('/report');
                const html = await response.text();
                document.getElementById('report-container').innerHTML = html;
                document.getElementById('last-updated').textContent = 'Last updated: ' + new Date().toLocaleString();
                updateStatus('Report loaded successfully', 'success');
            } catch (error) {
                updateStatus('Error loading report: ' + error.message, 'error');
            }
        }

        function refreshNow() {
            loadReport();
        }

        async function runNewTest() {
            if (!confirm('Run a new stress test? This will take about 30 seconds.')) return;

            try {
                updateStatus('Starting new stress test...', 'loading');
                await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 5, query_type: 'simple' })
                });
                updateStatus('Test completed, refreshing report...', 'loading');
                setTimeout(loadReport, 2000);
            } catch (error) {
                updateStatus('Error running test: ' + error.message, 'error');
            }
        }

        function toggleAutoRefresh() {
            autoRefresh = !autoRefresh;
            const btn = document.getElementById('toggle-btn');
            const status = document.getElementById('auto-refresh-status');

            if (autoRefresh) {
                btn.textContent = 'Pause';
                status.textContent = 'Auto-refresh: ON';
                startAutoRefresh();
            } else {
                btn.textContent = 'Resume';
                status.textContent = 'Auto-refresh: OFF';
                if (refreshInterval) clearInterval(refreshInterval);
            }
        }

        function startAutoRefresh() {
            if (refreshInterval) clearInterval(refreshInterval);
            refreshInterval = setInterval(() => {
                if (autoRefresh) loadReport();
            }, 30000); // Refresh every 30 seconds
        }

        // Initialize
        loadReport();
        startAutoRefresh();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// parseTestConfig parses test configuration from request
func (s *Server) parseTestConfig(r *http.Request) *stresstester.TestConfig {
	config := stresstester.DefaultTestConfig()

	// Parse query parameters or JSON body
	if r.Header.Get("Content-Type") == "application/json" {
		var reqConfig struct {
			Duration           int    `json:"duration"`
			Concurrency        int    `json:"concurrency"`
			QueryType          string `json:"query_type"`
			DelayBetween       int    `json:"delay_between"`
			OperationsLimit    int    `json:"operations_limit"`
			BurstMode          bool   `json:"burst_mode"`
			RandomizeQueries   bool   `json:"randomize_queries"`
			ReadPercent        int    `json:"read_percent"`
			WritePercent       int    `json:"write_percent"`
			TransactionPercent int    `json:"transaction_percent"`
		}

		if err := json.NewDecoder(r.Body).Decode(&reqConfig); err == nil {
			if reqConfig.Duration > 0 {
				config.Duration = time.Duration(reqConfig.Duration) * time.Second
			}
			if reqConfig.Concurrency > 0 {
				config.Concurrency = reqConfig.Concurrency
			}
			if reqConfig.QueryType != "" {
				config.QueryType = reqConfig.QueryType
			}
			if reqConfig.DelayBetween > 0 {
				config.DelayBetween = time.Duration(reqConfig.DelayBetween) * time.Millisecond
			}
			if reqConfig.OperationsLimit > 0 {
				config.OperationsLimit = reqConfig.OperationsLimit
			}
			config.BurstMode = reqConfig.BurstMode
			config.RandomizeQueries = reqConfig.RandomizeQueries

			// Set mixed ratio if provided
			if reqConfig.ReadPercent > 0 || reqConfig.WritePercent > 0 || reqConfig.TransactionPercent > 0 {
				config.MixedRatio.ReadPercent = reqConfig.ReadPercent
				config.MixedRatio.WritePercent = reqConfig.WritePercent
				config.MixedRatio.TransactionPercent = reqConfig.TransactionPercent
			}
		}
	} else {
		// Parse from query parameters
		if duration := r.URL.Query().Get("duration"); duration != "" {
			if d, err := strconv.Atoi(duration); err == nil && d > 0 {
				config.Duration = time.Duration(d) * time.Second
			}
		}
		if concurrency := r.URL.Query().Get("concurrency"); concurrency != "" {
			if c, err := strconv.Atoi(concurrency); err == nil && c > 0 {
				config.Concurrency = c
			}
		}
		if queryType := r.URL.Query().Get("query_type"); queryType != "" {
			config.QueryType = queryType
		}
		if burstMode := r.URL.Query().Get("burst_mode"); burstMode == "true" {
			config.BurstMode = true
		}
		if randomize := r.URL.Query().Get("randomize_queries"); randomize == "true" {
			config.RandomizeQueries = true
		}
		if readPercent := r.URL.Query().Get("read_percent"); readPercent != "" {
			if rp, err := strconv.Atoi(readPercent); err == nil && rp >= 0 && rp <= 100 {
				config.MixedRatio.ReadPercent = rp
			}
		}
		if writePercent := r.URL.Query().Get("write_percent"); writePercent != "" {
			if wp, err := strconv.Atoi(writePercent); err == nil && wp >= 0 && wp <= 100 {
				config.MixedRatio.WritePercent = wp
			}
		}
		if txnPercent := r.URL.Query().Get("transaction_percent"); txnPercent != "" {
			if tp, err := strconv.Atoi(txnPercent); err == nil && tp >= 0 && tp <= 100 {
				config.MixedRatio.TransactionPercent = tp
			}
		}
	}

	return config
}

// countConnected counts connected databases
func (s *Server) countConnected(status map[string]string) int {
	count := 0
	for _, s := range status {
		if s == "Connected" {
			count++
		}
	}
	return count
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

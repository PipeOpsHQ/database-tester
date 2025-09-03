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
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Courier New', monospace;
            background: #000;
            color: #00ff00;
            padding: 20px;
            line-height: 1.4;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background: #000;
            border: 1px solid #333;
            padding: 20px;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            border-bottom: 1px solid #333;
            padding-bottom: 15px;
        }
        h1 {
            color: #00ff00;
            font-size: 18px;
            font-weight: normal;
        }
        .subtitle {
            color: #888;
            font-size: 12px;
            margin-top: 5px;
        }
        .section {
            margin: 20px 0;
            border-top: 1px solid #333;
            padding-top: 15px;
        }
        .section-title {
            color: #fff;
            font-size: 14px;
            margin-bottom: 10px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .command-list {
            list-style: none;
            margin: 0;
            padding: 0;
        }
        .command-list li {
            margin: 5px 0;
            padding: 2px 0;
        }
        .command {
            color: #00ffff;
            cursor: pointer;
            text-decoration: none;
            font-weight: bold;
        }
        .command:hover {
            color: #fff;
            background: #333;
            padding: 2px 4px;
        }
        .description {
            color: #888;
            margin-left: 10px;
        }
        .status-line {
            color: #ffff00;
            font-size: 12px;
            padding: 10px;
            background: #111;
            border: 1px solid #333;
        }
        a.command {
            text-decoration: none;
        }
        a.command:visited {
            color: #00ffff;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Database Stress Test Service</h1>
            <div class="subtitle">PipeOps TCP/UDP Port Testing - Terminal Interface</div>
        </div>

        <div class="section">
            <div class="section-title">Quick Commands</div>
            <ul class="command-list">
                <li><span class="command" onclick="connectAll()">connect-all</span><span class="description">- Connect to all databases</span></li>
                <li><span class="command" onclick="runSimpleTest()">test-simple</span><span class="description">- Run simple stress test (30s)</span></li>
                <li><span class="command" onclick="runHeavyTest()">test-heavy</span><span class="description">- Run heavy load test (120s)</span></li>
                <li><span class="command" onclick="checkStatus()">status</span><span class="description">- Check database connections</span></li>
                <li><span class="command" onclick="disconnectAll()">disconnect-all</span><span class="description">- Disconnect all databases</span></li>
            </ul>
        </div>

        <div class="section">
            <div class="section-title">Reports & Configuration</div>
            <ul class="command-list">
                <li><a href="/report" class="command">view-report</a><span class="description">- View full test report</span></li>
                <li><a href="/live" class="command">live-report</a><span class="description">- Live updating report</span></li>
                <li><a href="/config-ui" class="command">configure</a><span class="description">- Database configuration</span></li>
            </ul>
        </div>

        <div class="section">
            <div class="section-title">API Endpoints</div>
            <ul class="command-list">
                <li><span class="command">GET /api/health</span><span class="description">- Service health check</span></li>
                <li><span class="command">GET /api/status</span><span class="description">- Connection status</span></li>
                <li><span class="command">POST /api/connect</span><span class="description">- Connect databases</span></li>
                <li><span class="command">POST /api/test</span><span class="description">- Run stress test</span></li>
            </ul>
        </div>

        <div class="section">
            <div class="section-title">System Status</div>
            <div id="status-output" class="status-line">Ready - Click commands above to interact</div>
        </div>
    </div>

    <script>
        function updateStatus(message) {
            const status = document.getElementById('status-output');
            status.textContent = new Date().toLocaleTimeString() + ' - ' + message;
        }

        async function connectAll() {
            updateStatus('Connecting to all databases...');
            try {
                const response = await fetch('/api/connect', { method: 'POST' });
                const data = await response.json();
                updateStatus('Connect attempt completed - ' + data.connected_count + ' databases connected');
            } catch (error) {
                updateStatus('Error: ' + error.message);
            }
        }

        async function runSimpleTest() {
            updateStatus('Starting simple stress test (30s)...');
            try {
                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 5, query_type: 'simple' })
                });
                const data = await response.json();
                updateStatus('Simple test completed - view report for details');
            } catch (error) {
                updateStatus('Error: ' + error.message);
            }
        }

        async function runHeavyTest() {
            updateStatus('Starting heavy load test (120s)...');
            try {
                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 120, concurrency: 20, query_type: 'heavy_read' })
                });
                const data = await response.json();
                updateStatus('Heavy test completed - view report for details');
            } catch (error) {
                updateStatus('Error: ' + error.message);
            }
        }

        async function checkStatus() {
            updateStatus('Checking database status...');
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                let connected = 0;
                let total = 0;
                for (const [db, status] of Object.entries(data.connection_status)) {
                    total++;
                    if (status === 'Connected') connected++;
                }
                updateStatus('Status check complete - ' + connected + '/' + total + ' databases connected');
            } catch (error) {
                updateStatus('Error: ' + error.message);
            }
        }

        async function disconnectAll() {
            updateStatus('Disconnecting from all databases...');
            try {
                const response = await fetch('/api/disconnect', { method: 'POST' });
                const data = await response.json();
                updateStatus('All databases disconnected');
            } catch (error) {
                updateStatus('Error: ' + error.message);
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
        body {
            font-family: 'Courier New', monospace;
            background: #000;
            color: #00ff00;
            padding: 20px;
            line-height: 1.4;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: #000;
            border: 1px solid #333;
            padding: 20px;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
            border-bottom: 1px solid #333;
            padding-bottom: 15px;
        }
        h1 {
            color: #00ff00;
            font-size: 18px;
            font-weight: normal;
        }
        .subtitle {
            color: #888;
            font-size: 12px;
            margin-top: 5px;
        }
        .nav-buttons {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 15px;
            border-bottom: 1px solid #333;
        }
        .nav-buttons a, .nav-buttons button {
            color: #00ffff;
            background: transparent;
            border: 1px solid #333;
            padding: 5px 15px;
            margin: 5px;
            text-decoration: none;
            font-family: inherit;
            cursor: pointer;
        }
        .nav-buttons a:hover, .nav-buttons button:hover {
            background: #333;
            color: #fff;
        }
        .database-card {
            border: 1px solid #333;
            margin-bottom: 20px;
            padding: 15px;
        }
        .card-header {
            border-bottom: 1px solid #333;
            padding-bottom: 10px;
            margin-bottom: 15px;
        }
        .card-title {
            color: #fff;
            font-size: 14px;
            text-transform: uppercase;
        }
        .status-indicator {
            float: right;
            font-size: 12px;
        }
        .status-connected { color: #00ff00; }
        .status-disconnected { color: #ff0000; }
        .status-unknown { color: #ffff00; }
        .form-group {
            margin-bottom: 10px;
        }
        .form-label {
            color: #888;
            font-size: 12px;
            margin-bottom: 3px;
            display: block;
        }
        .form-input {
            width: 100%;
            background: #111;
            border: 1px solid #333;
            color: #00ff00;
            padding: 5px;
            font-family: inherit;
            font-size: 12px;
        }
        .form-input:focus {
            outline: none;
            border-color: #00ffff;
        }
        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
        }
        .card-actions {
            margin-top: 15px;
            text-align: center;
        }
        .card-actions button {
            background: transparent;
            border: 1px solid #333;
            color: #00ffff;
            padding: 5px 10px;
            margin: 0 5px;
            cursor: pointer;
            font-family: inherit;
            font-size: 11px;
        }
        .card-actions button:hover {
            background: #333;
            color: #fff;
        }
        .message {
            padding: 10px;
            margin: 10px 0;
            border: 1px solid #333;
            font-size: 12px;
            display: none;
        }
        .message.success { color: #00ff00; border-color: #00ff00; }
        .message.error { color: #ff0000; border-color: #ff0000; }
        .message.info { color: #ffff00; border-color: #ffff00; }
        .loading {
            text-align: center;
            color: #ffff00;
            font-size: 12px;
            display: none;
        }
        .global-actions {
            border-top: 1px solid #333;
            padding-top: 20px;
            text-align: center;
        }
        .global-actions h3 {
            color: #fff;
            font-size: 14px;
            margin-bottom: 15px;
            text-transform: uppercase;
        }
        .global-actions button {
            background: transparent;
            border: 1px solid #333;
            color: #00ffff;
            padding: 8px 15px;
            margin: 5px;
            cursor: pointer;
            font-family: inherit;
        }
        .global-actions button:hover {
            background: #333;
            color: #fff;
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
            <a href="/">dashboard</a>
            <button onclick="loadCurrentConfig()">reload-config</button>
            <button onclick="saveAllConfigs()">save-all</button>
            <button onclick="testAllConnections()">test-all</button>
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
            <button onclick="connectAllDatabases()">connect-all</button>
            <button onclick="runQuickStressTest()">test-simple</button>
            <button onclick="viewReport()">view-report</button>
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
            showMessage('Starting simple test (30s)...', 'info');
            try {
                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 5, query_type: 'simple' })
                });
                showMessage('Simple test completed - check report', 'success');
            } catch (error) {
                showMessage('Error: ' + error.message, 'error');
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
        body {
            font-family: 'Courier New', monospace;
            margin: 40px;
            text-align: center;
            background: #000;
            color: #00ff00;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            background: #000;
            padding: 40px;
            border: 1px solid #333;
        }
        .btn {
            padding: 8px 16px;
            border: 1px solid #333;
            background: transparent;
            cursor: pointer;
            text-decoration: none;
            font-size: 12px;
            color: #00ffff;
            margin: 10px;
            font-family: inherit;
        }
        .btn:hover {
            background: #333;
            color: #fff;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>no test results available</h1>
        <p>no stress tests have been run yet - run a test to see the report</p>
        <a href="/" class="btn">dashboard</a>
        <button onclick="runTest()" class="btn">run-test-now</button>
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
        body {
            font-family: 'Courier New', monospace;
            margin: 0;
            padding: 20px;
            background: #000;
            color: #00ff00;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            border: 1px solid #333;
            background: #000;
        }
        .header {
            background: #000;
            color: #00ff00;
            padding: 15px;
            border-bottom: 1px solid #333;
            text-align: center;
        }
        .header h1 {
            font-size: 16px;
            font-weight: normal;
            margin: 0;
        }
        .controls {
            background: #000;
            padding: 10px;
            border-bottom: 1px solid #333;
            text-align: center;
        }
        .btn {
            padding: 5px 10px;
            margin: 0 5px;
            border: 1px solid #333;
            background: transparent;
            color: #00ffff;
            cursor: pointer;
            font-family: inherit;
            font-size: 11px;
        }
        .btn:hover {
            background: #333;
            color: #fff;
        }
        .status {
            margin: 10px;
            padding: 8px;
            border: 1px solid #333;
            font-size: 12px;
        }
        .status.loading { color: #ffff00; }
        .status.success { color: #00ff00; }
        .status.error { color: #ff0000; }
        #report-container {
            background: #000;
            padding: 15px;
            font-size: 12px;
            overflow: auto;
        }
        #last-updated {
            font-size: 12px;
            color: #888;
        }
        #auto-refresh-status {
            color: #888;
            font-size: 11px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Live Database Stress Test Report</h1>
            <div id="last-updated">Loading...</div>
        </div>

        <div class="controls">
            <button onclick="refreshNow()" class="btn">refresh-now</button>
            <button onclick="runNewTest()" class="btn">run-test</button>
            <span id="auto-refresh-status">auto-refresh: on</span>
            <button onclick="toggleAutoRefresh()" class="btn" id="toggle-btn">pause</button>
        </div>

        <div id="status" class="status">ready - loading report...</div>
        <div id="report-container">loading...</div>
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
            try {
                updateStatus('starting new test (30s)...', 'loading');
                await fetch('/api/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ duration: 30, concurrency: 5, query_type: 'simple' })
                });
                updateStatus('test completed, refreshing...', 'loading');
                setTimeout(loadReport, 2000);
            } catch (error) {
                updateStatus('error: ' + error.message, 'error');
            }
        }

        function toggleAutoRefresh() {
            autoRefresh = !autoRefresh;
            const btn = document.getElementById('toggle-btn');
            const status = document.getElementById('auto-refresh-status');

            if (autoRefresh) {
                btn.textContent = 'pause';
                status.textContent = 'auto-refresh: on';
                startAutoRefresh();
            } else {
                btn.textContent = 'resume';
                status.textContent = 'auto-refresh: off';
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

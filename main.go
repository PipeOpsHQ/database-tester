package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"tcp-stress-test/config"
	"tcp-stress-test/server"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Database Stress Test Service...")

	// Load configuration
	cfg := config.LoadConfig()

	// Print startup information
	printStartupInfo(cfg)

	// Create and configure server
	srv := server.NewServer(cfg)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutdown signal received, stopping server...")
		srv.Stop()
		os.Exit(0)
	}()

	// Start server
	log.Printf("Service ready!")
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// printStartupInfo displays configuration and startup information
func printStartupInfo(cfg *config.DatabaseConfig) {
	fmt.Println()
	fmt.Println("===============================================")
	fmt.Println("   Database Stress Test Service")
	fmt.Println("   PipeOps TCP/UDP Port Testing")
	fmt.Println("===============================================")
	fmt.Printf("Server: http://%s:%s\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Dashboard: http://%s:%s/\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Reports: http://%s:%s/report\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Live Report: http://%s:%s/live\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Health Check: http://%s:%s/api/health\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println()
	fmt.Println("Database Connections:")
	fmt.Printf("   • MySQL:      %s:%s/%s\n", cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)
	fmt.Printf("   • PostgreSQL: %s:%s/%s\n", cfg.PostgreSQL.Host, cfg.PostgreSQL.Port, cfg.PostgreSQL.Database)
	fmt.Printf("   • MongoDB:    %s/%s\n", cfg.MongoDB.URI, cfg.MongoDB.Database)
	fmt.Printf("   • Redis:      %s:%s (DB:%d)\n", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
	fmt.Printf("   • MSSQL:      %s:%s/%s\n", cfg.MSSQL.Host, cfg.MSSQL.Port, cfg.MSSQL.Database)
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("   • Multi-database stress testing")
	fmt.Println("   • Complex query benchmarking")
	fmt.Println("   • Real-time HTML reports")
	fmt.Println("   • RESTful API")
	fmt.Println("   • Live dashboard")
	fmt.Println("   • Connection monitoring")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("   Set these to customize database connections:")
	fmt.Println("   MYSQL_HOST, MYSQL_PORT, MYSQL_USERNAME, MYSQL_PASSWORD, MYSQL_DATABASE")
	fmt.Println("   POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USERNAME, POSTGRES_PASSWORD, POSTGRES_DATABASE")
	fmt.Println("   MONGO_URI, MONGO_DATABASE")
	fmt.Println("   REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, REDIS_DB")
	fmt.Println("   MSSQL_HOST, MSSQL_PORT, MSSQL_USERNAME, MSSQL_PASSWORD, MSSQL_DATABASE")
	fmt.Println("   SERVER_HOST, SERVER_PORT")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("   1. Visit the dashboard to see connection status")
	fmt.Println("   2. Click 'Connect All DBs' to establish connections")
	fmt.Println("   3. Run 'Run Stress Test' to start testing")
	fmt.Println("   4. View results in the report page")
	fmt.Println()
	fmt.Println("API Endpoints:")
	fmt.Println("   GET  /api/health - Service health check")
	fmt.Println("   GET  /api/status - Database connection status")
	fmt.Println("   POST /api/connect - Connect to all databases")
	fmt.Println("   POST /api/test - Run stress test on all DBs")
	fmt.Println("   POST /api/test/{database} - Test specific database")
	fmt.Println("   POST /api/benchmark - Custom query benchmark")
	fmt.Println()
	fmt.Println("===============================================")
	fmt.Println()
}

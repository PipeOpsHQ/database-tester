package config

import (
	"os"
	"strconv"
	"sync"
)

// DatabaseConfig holds configuration for all database connections
type DatabaseConfig struct {
	MySQL      MySQLConfig
	PostgreSQL PostgreSQLConfig
	MongoDB    MongoDBConfig
	Redis      RedisConfig
	MSSQL      MSSQLConfig
	Server     ServerConfig
	mutex      sync.RWMutex // For thread-safe updates
}

// DynamicDatabaseConfig holds runtime updatable database configurations
type DynamicDatabaseConfig struct {
	MySQL      *MySQLConfig      `json:"mysql,omitempty"`
	PostgreSQL *PostgreSQLConfig `json:"postgresql,omitempty"`
	MongoDB    *MongoDBConfig    `json:"mongodb,omitempty"`
	Redis      *RedisConfig      `json:"redis,omitempty"`
	MSSQL      *MSSQLConfig      `json:"mssql,omitempty"`
}

// MySQLConfig holds MySQL connection configuration
type MySQLConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	DSN      string
}

// PostgreSQLConfig holds PostgreSQL connection configuration
type PostgreSQLConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	SSLMode  string
	DSN      string
}

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI      string
	Database string
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// MSSQLConfig holds MSSQL connection configuration
type MSSQLConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	Database string
	DSN      string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Host string
}

// ConfigManager holds the global configuration instance
var configManager *DatabaseConfig
var configOnce sync.Once

// LoadConfig loads configuration from environment variables
func LoadConfig() *DatabaseConfig {
	config := &DatabaseConfig{
		MySQL: MySQLConfig{
			Host:     getEnv("MYSQL_HOST", "localhost"),
			Port:     getEnv("MYSQL_PORT", "3306"),
			Username: getEnv("MYSQL_USERNAME", "root"),
			Password: getEnv("MYSQL_PASSWORD", "password"),
			Database: getEnv("MYSQL_DATABASE", "testdb"),
		},
		PostgreSQL: PostgreSQLConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			Username: getEnv("POSTGRES_USERNAME", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "password"),
			Database: getEnv("POSTGRES_DATABASE", "testdb"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGO_URI", "mongodb://root:StrongPassword123!@localhost:27017"),
			Database: getEnv("MONGO_DATABASE", "testdb"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		MSSQL: MSSQLConfig{
			Host:     getEnv("MSSQL_HOST", "localhost"),
			Port:     getEnv("MSSQL_PORT", "1433"),
			Username: getEnv("MSSQL_USERNAME", "sa"),
			Password: getEnv("MSSQL_PASSWORD", "Password123!"),
			Database: getEnv("MSSQL_DATABASE", "testdb"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
	}

	// Build DSN strings
	config.MySQL.DSN = config.MySQL.Username + ":" + config.MySQL.Password + "@tcp(" + config.MySQL.Host + ":" + config.MySQL.Port + ")/" + config.MySQL.Database + "?charset=utf8mb4&parseTime=True&loc=Local"

	config.PostgreSQL.DSN = "host=" + config.PostgreSQL.Host + " port=" + config.PostgreSQL.Port + " user=" + config.PostgreSQL.Username + " password=" + config.PostgreSQL.Password + " dbname=" + config.PostgreSQL.Database + " sslmode=" + config.PostgreSQL.SSLMode

	config.MSSQL.DSN = "server=" + config.MSSQL.Host + ";user id=" + config.MSSQL.Username + ";password=" + config.MSSQL.Password + ";port=" + config.MSSQL.Port + ";database=" + config.MSSQL.Database + ";encrypt=disable"

	// Set as global config manager
	configOnce.Do(func() {
		configManager = config
	})

	return config
}

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *DatabaseConfig {
	if configManager == nil {
		return LoadConfig()
	}
	return configManager
}

// UpdateDatabaseConfig updates database configuration at runtime
func (dc *DatabaseConfig) UpdateDatabaseConfig(updates DynamicDatabaseConfig) error {
	dc.mutex.Lock()
	defer dc.mutex.Unlock()

	// Update MySQL config if provided
	if updates.MySQL != nil {
		dc.MySQL = *updates.MySQL
		dc.MySQL.DSN = dc.MySQL.Username + ":" + dc.MySQL.Password + "@tcp(" + dc.MySQL.Host + ":" + dc.MySQL.Port + ")/" + dc.MySQL.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// Update PostgreSQL config if provided
	if updates.PostgreSQL != nil {
		dc.PostgreSQL = *updates.PostgreSQL
		dc.PostgreSQL.DSN = "host=" + dc.PostgreSQL.Host + " port=" + dc.PostgreSQL.Port + " user=" + dc.PostgreSQL.Username + " password=" + dc.PostgreSQL.Password + " dbname=" + dc.PostgreSQL.Database + " sslmode=" + dc.PostgreSQL.SSLMode
	}

	// Update MongoDB config if provided
	if updates.MongoDB != nil {
		dc.MongoDB = *updates.MongoDB
	}

	// Update Redis config if provided
	if updates.Redis != nil {
		dc.Redis = *updates.Redis
	}

	// Update MSSQL config if provided
	if updates.MSSQL != nil {
		dc.MSSQL = *updates.MSSQL
		dc.MSSQL.DSN = "server=" + dc.MSSQL.Host + ";user id=" + dc.MSSQL.Username + ";password=" + dc.MSSQL.Password + ";port=" + dc.MSSQL.Port + ";database=" + dc.MSSQL.Database + ";encrypt=disable"
	}

	return nil
}

// GetDatabaseConfig returns current database configuration (thread-safe)
func (dc *DatabaseConfig) GetDatabaseConfig() DynamicDatabaseConfig {
	dc.mutex.RLock()
	defer dc.mutex.RUnlock()

	return DynamicDatabaseConfig{
		MySQL:      &dc.MySQL,
		PostgreSQL: &dc.PostgreSQL,
		MongoDB:    &dc.MongoDB,
		Redis:      &dc.Redis,
		MSSQL:      &dc.MSSQL,
	}
}

// TestDatabaseConnection tests a database connection with provided config
func TestDatabaseConnection(dbType string, config interface{}) error {
	switch dbType {
	case "mysql":
		if cfg, ok := config.(*MySQLConfig); ok {
			return testMySQLConnection(cfg)
		}
	case "postgresql":
		if cfg, ok := config.(*PostgreSQLConfig); ok {
			return testPostgreSQLConnection(cfg)
		}
	case "redis":
		if cfg, ok := config.(*RedisConfig); ok {
			return testRedisConnection(cfg)
		}
	case "mssql":
		if cfg, ok := config.(*MSSQLConfig); ok {
			return testMSSQLConnection(cfg)
		}
	case "mongodb":
		if cfg, ok := config.(*MongoDBConfig); ok {
			return testMongoDBConnection(cfg)
		}
	}
	return nil
}

// Helper functions for testing connections
func testMySQLConnection(cfg *MySQLConfig) error {
	// This will be implemented by importing the connector temporarily
	return nil
}

func testPostgreSQLConnection(cfg *PostgreSQLConfig) error {
	// This will be implemented by importing the connector temporarily
	return nil
}

func testRedisConnection(cfg *RedisConfig) error {
	// This will be implemented by importing the connector temporarily
	return nil
}

func testMSSQLConnection(cfg *MSSQLConfig) error {
	// This will be implemented by importing the connector temporarily
	return nil
}

func testMongoDBConnection(cfg *MongoDBConfig) error {
	// This will be implemented by importing the connector temporarily
	return nil
}

// getEnv gets an environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as integer or returns default value
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"tcp-stress-test/config"
	"tcp-stress-test/queries"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DatabaseConnector defines the interface for database operations
type DatabaseConnector interface {
	Connect() error
	Disconnect() error
	ExecuteQuery(query string) (*QueryResult, error)
	ExecuteComplexQuery() (*QueryResult, error)
	GetConnectionInfo() string
	IsConnected() bool
	TestConnection() error
}

// QueryResult holds the result of a database query
type QueryResult struct {
	Duration     time.Duration
	RowsAffected int64
	Error        error
	Success      bool
	Message      string
}

// MySQLConnector implements DatabaseConnector for MySQL
type MySQLConnector struct {
	config   *config.MySQLConfig
	db       *sql.DB
	queryLib *queries.QueryLibrary
}

// PostgreSQLConnector implements DatabaseConnector for PostgreSQL
type PostgreSQLConnector struct {
	config   *config.PostgreSQLConfig
	db       *sql.DB
	queryLib *queries.QueryLibrary
}

// MongoDBConnector implements DatabaseConnector for MongoDB
type MongoDBConnector struct {
	config   *config.MongoDBConfig
	client   *mongo.Client
	db       *mongo.Database
	queryLib *queries.QueryLibrary
}

// RedisConnector implements DatabaseConnector for Redis
type RedisConnector struct {
	config   *config.RedisConfig
	client   *redis.Client
	queryLib *queries.QueryLibrary
}

// MSSQLConnector implements DatabaseConnector for MSSQL
type MSSQLConnector struct {
	config   *config.MSSQLConfig
	db       *sql.DB
	queryLib *queries.QueryLibrary
}

// NewMySQLConnector creates a new MySQL connector
func NewMySQLConnector(cfg *config.MySQLConfig) *MySQLConnector {
	return &MySQLConnector{
		config:   cfg,
		queryLib: queries.NewQueryLibrary(),
	}
}

// Connect establishes MySQL connection
func (m *MySQLConnector) Connect() error {
	var err error
	m.db, err = sql.Open("mysql", m.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL: %v", err)
	}

	m.db.SetMaxOpenConns(25)
	m.db.SetMaxIdleConns(5)
	m.db.SetConnMaxLifetime(5 * time.Minute)

	return m.db.Ping()
}

// Disconnect closes MySQL connection
func (m *MySQLConnector) Disconnect() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// ExecuteQuery executes a query on MySQL
func (m *MySQLConnector) ExecuteQuery(query string) (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	rows, err := m.db.Query(query)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}
	defer rows.Close()

	var count int64 = 0
	for rows.Next() {
		count++
	}

	result.Duration = time.Since(start)
	result.RowsAffected = count
	result.Success = true
	result.Message = fmt.Sprintf("Query executed successfully, %d rows returned", count)

	return result, nil
}

// ExecuteComplexQuery executes a complex query for stress testing
func (m *MySQLConnector) ExecuteComplexQuery() (*QueryResult, error) {
	// Get a complex query from the query library
	query, err := m.queryLib.GetQuery(queries.DatabaseMySQL, queries.QueryTypeComplex)
	if err != nil {
		// Fallback to analytics query if complex not available
		query, err = m.queryLib.GetQuery(queries.DatabaseMySQL, queries.QueryTypeAnalytics)
		if err != nil {
			// Final fallback to join query
			query, err = m.queryLib.GetQuery(queries.DatabaseMySQL, queries.QueryTypeJoin)
			if err != nil {
				// Ultimate fallback to hardcoded query
				complexQuery := `
					SELECT
						t1.id,
						t1.name,
						COUNT(t2.id) as related_count,
						AVG(t2.value) as avg_value,
						MAX(t2.created_at) as latest_update
					FROM
						(SELECT 1 as id, 'Test Record 1' as name UNION ALL
						 SELECT 2 as id, 'Test Record 2' as name UNION ALL
						 SELECT 3 as id, 'Test Record 3' as name) t1
					LEFT JOIN
						(SELECT 1 as id, 1 as parent_id, RAND() * 100 as value, NOW() as created_at FROM DUAL UNION ALL
						 SELECT 2 as id, 1 as parent_id, RAND() * 100 as value, NOW() as created_at FROM DUAL UNION ALL
						 SELECT 3 as id, 2 as parent_id, RAND() * 100 as value, NOW() as created_at FROM DUAL UNION ALL
						 SELECT 4 as id, 3 as parent_id, RAND() * 100 as value, NOW() as created_at FROM DUAL) t2
					ON t1.id = t2.parent_id
					GROUP BY t1.id, t1.name
					ORDER BY t1.id
				`
				return m.ExecuteQuery(complexQuery)
			}
		}
	}

	log.Printf("Executing MySQL complex query: %s", query.Name)
	return m.ExecuteQuery(query.SQL)
}

// GetConnectionInfo returns MySQL connection information
func (m *MySQLConnector) GetConnectionInfo() string {
	return fmt.Sprintf("MySQL: %s:%s/%s", m.config.Host, m.config.Port, m.config.Database)
}

// IsConnected checks if MySQL is connected
func (m *MySQLConnector) IsConnected() bool {
	if m.db == nil {
		return false
	}
	return m.db.Ping() == nil
}

// TestConnection tests the MySQL connection
func (m *MySQLConnector) TestConnection() error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.Ping()
}

// UpdateConfig updates the MySQL configuration and reconnects if necessary
func (m *MySQLConnector) UpdateConfig(cfg *config.MySQLConfig) error {
	// Disconnect existing connection
	if m.db != nil {
		m.db.Close()
		m.db = nil
	}

	// Update configuration
	m.config = cfg

	// Reconnect with new configuration
	return m.Connect()
}

// NewPostgreSQLConnector creates a new PostgreSQL connector
func NewPostgreSQLConnector(cfg *config.PostgreSQLConfig) *PostgreSQLConnector {
	return &PostgreSQLConnector{
		config:   cfg,
		queryLib: queries.NewQueryLibrary(),
	}
}

// Connect establishes PostgreSQL connection
func (p *PostgreSQLConnector) Connect() error {
	var err error
	p.db, err = sql.Open("postgres", p.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	p.db.SetMaxOpenConns(25)
	p.db.SetMaxIdleConns(5)
	p.db.SetConnMaxLifetime(5 * time.Minute)

	return p.db.Ping()
}

// Disconnect closes PostgreSQL connection
func (p *PostgreSQLConnector) Disconnect() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// ExecuteQuery executes a query on PostgreSQL
func (p *PostgreSQLConnector) ExecuteQuery(query string) (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	rows, err := p.db.Query(query)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}
	defer rows.Close()

	var count int64 = 0
	for rows.Next() {
		count++
	}

	result.Duration = time.Since(start)
	result.RowsAffected = count
	result.Success = true
	result.Message = fmt.Sprintf("Query executed successfully, %d rows returned", count)

	return result, nil
}

// ExecuteComplexQuery executes a complex query for stress testing
func (p *PostgreSQLConnector) ExecuteComplexQuery() (*QueryResult, error) {
	// Get a complex query from the query library
	query, err := p.queryLib.GetQuery(queries.DatabasePostgreSQL, queries.QueryTypeAnalytics)
	if err != nil {
		// Fallback to recursive query
		query, err = p.queryLib.GetQuery(queries.DatabasePostgreSQL, queries.QueryTypeRecursive)
		if err != nil {
			// Final fallback to complex query
			query, err = p.queryLib.GetQuery(queries.DatabasePostgreSQL, queries.QueryTypeComplex)
			if err != nil {
				// Ultimate fallback to hardcoded query
				complexQuery := `
					WITH test_data AS (
						SELECT
							generate_series(1, 1000) as id,
							'Record ' || generate_series(1, 1000) as name,
							random() * 100 as value,
							NOW() - (random() * INTERVAL '365 days') as created_at
					),
					aggregated AS (
						SELECT
							id,
							name,
							value,
							created_at,
							ROW_NUMBER() OVER (ORDER BY value DESC) as rank,
							AVG(value) OVER (PARTITION BY (id % 10)) as group_avg
						FROM test_data
					)
					SELECT
						rank,
						name,
						value,
						group_avg,
						created_at,
						CASE
							WHEN value > group_avg THEN 'Above Average'
							ELSE 'Below Average'
						END as performance
					FROM aggregated
					WHERE rank <= 50
					ORDER BY rank
				`
				return p.ExecuteQuery(complexQuery)
			}
		}
	}

	log.Printf("Executing PostgreSQL complex query: %s", query.Name)
	return p.ExecuteQuery(query.SQL)
}

// GetConnectionInfo returns PostgreSQL connection information
func (p *PostgreSQLConnector) GetConnectionInfo() string {
	return fmt.Sprintf("PostgreSQL: %s:%s/%s", p.config.Host, p.config.Port, p.config.Database)
}

// IsConnected checks if PostgreSQL is connected
func (p *PostgreSQLConnector) IsConnected() bool {
	if p.db == nil {
		return false
	}
	return p.db.Ping() == nil
}

// TestConnection tests the PostgreSQL connection
func (p *PostgreSQLConnector) TestConnection() error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}
	return p.db.Ping()
}

// UpdateConfig updates the PostgreSQL configuration and reconnects if necessary
func (p *PostgreSQLConnector) UpdateConfig(cfg *config.PostgreSQLConfig) error {
	// Disconnect existing connection
	if p.db != nil {
		p.db.Close()
		p.db = nil
	}

	// Update configuration
	p.config = cfg

	// Reconnect with new configuration
	return p.Connect()
}

// NewMongoDBConnector creates a new MongoDB connector
func NewMongoDBConnector(cfg *config.MongoDBConfig) *MongoDBConnector {
	return &MongoDBConnector{
		config:   cfg,
		queryLib: queries.NewQueryLibrary(),
	}
}

// Connect establishes MongoDB connection
func (m *MongoDBConnector) Connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	m.client, err = mongo.Connect(ctx, options.Client().ApplyURI(m.config.URI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = m.client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	m.db = m.client.Database(m.config.Database)
	return nil
}

// Disconnect closes MongoDB connection
func (m *MongoDBConnector) Disconnect() error {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return m.client.Disconnect(ctx)
	}
	return nil
}

// ExecuteQuery executes a query on MongoDB
func (m *MongoDBConnector) ExecuteQuery(query string) (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := m.db.Collection("testcollection")

	// Simple find operation
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}
	defer cursor.Close(ctx)

	var count int64 = 0
	for cursor.Next(ctx) {
		count++
	}

	result.Duration = time.Since(start)
	result.RowsAffected = count
	result.Success = true
	result.Message = fmt.Sprintf("Query executed successfully, %d documents returned", count)

	return result, nil
}

// ExecuteComplexQuery executes a complex aggregation pipeline
func (m *MongoDBConnector) ExecuteComplexQuery() (*QueryResult, error) {
	// Get a complex query from the query library
	query, err := m.queryLib.GetQuery(queries.DatabaseMongoDB, queries.QueryTypeComplex)
	if err != nil {
		// Fallback to aggregation pipeline
		return m.executeDefaultAggregation()
	}

	log.Printf("Executing MongoDB complex query: %s", query.Name)
	return m.executeAggregationPipeline()
}

// executeDefaultAggregation executes a default aggregation pipeline
func (m *MongoDBConnector) executeDefaultAggregation() (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := m.db.Collection("testcollection")

	// Complex aggregation pipeline
	pipeline := []bson.M{
		{"$addFields": bson.M{
			"randomValue": bson.M{"$rand": bson.M{}},
			"category":    bson.M{"$mod": []interface{}{bson.M{"$toInt": "$_id"}, 5}},
		}},
		{"$group": bson.M{
			"_id":      "$category",
			"count":    bson.M{"$sum": 1},
			"avgValue": bson.M{"$avg": "$randomValue"},
			"maxValue": bson.M{"$max": "$randomValue"},
			"minValue": bson.M{"$min": "$randomValue"},
		}},
		{"$sort": bson.M{"_id": 1}},
		{"$project": bson.M{
			"category": "$_id",
			"count":    1,
			"avgValue": 1,
			"maxValue": 1,
			"minValue": 1,
			"range":    bson.M{"$subtract": []interface{}{"$maxValue", "$minValue"}},
			"_id":      0,
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}
	defer cursor.Close(ctx)

	var count int64 = 0
	for cursor.Next(ctx) {
		count++
	}

	result.Duration = time.Since(start)
	result.RowsAffected = count
	result.Success = true
	result.Message = fmt.Sprintf("Aggregation executed successfully, %d groups returned", count)

	return result, nil
}

// executeAggregationPipeline executes aggregation pipeline
func (m *MongoDBConnector) executeAggregationPipeline() (*QueryResult, error) {
	return m.executeDefaultAggregation()
}

// GetConnectionInfo returns MongoDB connection information
func (m *MongoDBConnector) GetConnectionInfo() string {
	return fmt.Sprintf("MongoDB: %s/%s", m.config.URI, m.config.Database)
}

// IsConnected checks if MongoDB is connected
func (m *MongoDBConnector) IsConnected() bool {
	if m.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Ping(ctx, nil) == nil
}

// TestConnection tests the MongoDB connection
func (m *MongoDBConnector) TestConnection() error {
	if m.client == nil {
		return fmt.Errorf("database not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Ping(ctx, nil)
}

// UpdateConfig updates the MongoDB configuration and reconnects if necessary
func (m *MongoDBConnector) UpdateConfig(cfg *config.MongoDBConfig) error {
	// Disconnect existing connection
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		m.client.Disconnect(ctx)
		m.client = nil
		m.db = nil
	}

	// Update configuration
	m.config = cfg

	// Reconnect with new configuration
	return m.Connect()
}

// NewRedisConnector creates a new Redis connector
func NewRedisConnector(cfg *config.RedisConfig) *RedisConnector {
	return &RedisConnector{
		config:   cfg,
		queryLib: queries.NewQueryLibrary(),
	}
}

// Connect establishes Redis connection
func (r *RedisConnector) Connect() error {
	r.client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", r.config.Host, r.config.Port),
		Password: r.config.Password,
		DB:       r.config.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.client.Ping(ctx).Result()
	return err
}

// Disconnect closes Redis connection
func (r *RedisConnector) Disconnect() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// ExecuteQuery executes a query on Redis
func (r *RedisConnector) ExecuteQuery(query string) (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute a simple GET operation
	val, err := r.client.Get(ctx, "test-key").Result()
	if err != nil && err != redis.Nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}

	result.Duration = time.Since(start)
	result.RowsAffected = 1
	result.Success = true
	if err == redis.Nil {
		result.Message = "Key not found"
	} else {
		result.Message = fmt.Sprintf("Value retrieved: %s", val)
	}

	return result, nil
}

// ExecuteComplexQuery executes complex Redis operations
func (r *RedisConnector) ExecuteComplexQuery() (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Complex Redis operations
	pipe := r.client.Pipeline()

	// Set multiple keys
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("stress-test-key-%d", i)
		value := fmt.Sprintf("value-%d-%d", i, time.Now().Unix())
		pipe.Set(ctx, key, value, time.Hour)
	}

	// Execute pipeline
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}

	// Get all keys back
	keys, err := r.client.Keys(ctx, "stress-test-key-*").Result()
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}

	result.Duration = time.Since(start)
	result.RowsAffected = int64(len(cmds))
	result.Success = true
	result.Message = fmt.Sprintf("Pipeline executed: %d operations, %d keys found", len(cmds), len(keys))

	return result, nil
}

// GetConnectionInfo returns Redis connection information
func (r *RedisConnector) GetConnectionInfo() string {
	return fmt.Sprintf("Redis: %s:%s (DB:%d)", r.config.Host, r.config.Port, r.config.DB)
}

// IsConnected checks if Redis is connected
func (r *RedisConnector) IsConnected() bool {
	if r.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Ping(ctx).Result()
	return err == nil
}

// TestConnection tests the Redis connection
func (r *RedisConnector) TestConnection() error {
	if r.client == nil {
		return fmt.Errorf("Redis client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Ping(ctx).Result()
	return err
}

// UpdateConfig updates the Redis configuration and reconnects if necessary
func (r *RedisConnector) UpdateConfig(cfg *config.RedisConfig) error {
	// Disconnect existing connection
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}

	// Update configuration
	r.config = cfg

	// Reconnect with new configuration
	return r.Connect()
}

// NewMSSQLConnector creates a new MSSQL connector
func NewMSSQLConnector(cfg *config.MSSQLConfig) *MSSQLConnector {
	return &MSSQLConnector{
		config:   cfg,
		queryLib: queries.NewQueryLibrary(),
	}
}

// Connect establishes MSSQL connection
func (m *MSSQLConnector) Connect() error {
	var err error
	m.db, err = sql.Open("sqlserver", m.config.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to MSSQL: %v", err)
	}

	m.db.SetMaxOpenConns(25)
	m.db.SetMaxIdleConns(5)
	m.db.SetConnMaxLifetime(5 * time.Minute)

	return m.db.Ping()
}

// Disconnect closes MSSQL connection
func (m *MSSQLConnector) Disconnect() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// ExecuteQuery executes a query on MSSQL
func (m *MSSQLConnector) ExecuteQuery(query string) (*QueryResult, error) {
	start := time.Now()
	result := &QueryResult{}

	rows, err := m.db.Query(query)
	if err != nil {
		result.Error = err
		result.Success = false
		result.Duration = time.Since(start)
		return result, err
	}
	defer rows.Close()

	var count int64 = 0
	for rows.Next() {
		count++
	}

	result.Duration = time.Since(start)
	result.RowsAffected = count
	result.Success = true
	result.Message = fmt.Sprintf("Query executed successfully, %d rows returned", count)

	return result, nil
}

// ExecuteComplexQuery executes a complex query for stress testing
func (m *MSSQLConnector) ExecuteComplexQuery() (*QueryResult, error) {
	// Get a complex query from the query library
	query, err := m.queryLib.GetQuery(queries.DatabaseMSSQL, queries.QueryTypeAnalytics)
	if err != nil {
		// Fallback to recursive query
		query, err = m.queryLib.GetQuery(queries.DatabaseMSSQL, queries.QueryTypeRecursive)
		if err != nil {
			// Ultimate fallback to hardcoded query
			complexQuery := `
				WITH NumberSequence AS (
					SELECT 1 as num
					UNION ALL
					SELECT num + 1
					FROM NumberSequence
					WHERE num < 1000
				),
				TestData AS (
					SELECT
						num as id,
						'Record ' + CAST(num as VARCHAR(10)) as name,
						RAND(CHECKSUM(NEWID())) * 100 as value,
						DATEADD(day, -RAND(CHECKSUM(NEWID())) * 365, GETDATE()) as created_at
					FROM NumberSequence
				),
				RankedData AS (
					SELECT
						id,
						name,
						value,
						created_at,
						ROW_NUMBER() OVER (ORDER BY value DESC) as rank,
						AVG(value) OVER (PARTITION BY (id % 10)) as group_avg
					FROM TestData
				)
				SELECT TOP 50
					rank,
					name,
					value,
					group_avg,
					created_at,
					CASE
						WHEN value > group_avg THEN 'Above Average'
						ELSE 'Below Average'
					END as performance
				FROM RankedData
				ORDER BY rank
				OPTION (MAXRECURSION 0)
			`
			return m.ExecuteQuery(complexQuery)
		}
	}

	log.Printf("Executing MSSQL complex query: %s", query.Name)
	return m.ExecuteQuery(query.SQL)
}

// GetConnectionInfo returns MSSQL connection information
func (m *MSSQLConnector) GetConnectionInfo() string {
	return fmt.Sprintf("MSSQL: %s:%s/%s", m.config.Host, m.config.Port, m.config.Database)
}

// IsConnected checks if MSSQL is connected
func (m *MSSQLConnector) IsConnected() bool {
	if m.db == nil {
		return false
	}
	return m.db.Ping() == nil
}

// TestConnection tests the MSSQL connection
func (m *MSSQLConnector) TestConnection() error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.Ping()
}

// UpdateConfig updates the MSSQL configuration and reconnects if necessary
func (m *MSSQLConnector) UpdateConfig(cfg *config.MSSQLConfig) error {
	// Disconnect existing connection
	if m.db != nil {
		m.db.Close()
		m.db = nil
	}

	// Update configuration
	m.config = cfg

	// Reconnect with new configuration
	return m.Connect()
}

// ConnectorManager manages all database connectors
type ConnectorManager struct {
	MySQL      *MySQLConnector
	PostgreSQL *PostgreSQLConnector
	MongoDB    *MongoDBConnector
	Redis      *RedisConnector
	MSSQL      *MSSQLConnector
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(cfg *config.DatabaseConfig) *ConnectorManager {
	return &ConnectorManager{
		MySQL:      NewMySQLConnector(&cfg.MySQL),
		PostgreSQL: NewPostgreSQLConnector(&cfg.PostgreSQL),
		MongoDB:    NewMongoDBConnector(&cfg.MongoDB),
		Redis:      NewRedisConnector(&cfg.Redis),
		MSSQL:      NewMSSQLConnector(&cfg.MSSQL),
	}
}

// ConnectAll connects to all databases
func (cm *ConnectorManager) ConnectAll() {
	databases := []struct {
		name      string
		connector DatabaseConnector
	}{
		{"MySQL", cm.MySQL},
		{"PostgreSQL", cm.PostgreSQL},
		{"MongoDB", cm.MongoDB},
		{"Redis", cm.Redis},
		{"MSSQL", cm.MSSQL},
	}

	for _, db := range databases {
		if err := db.connector.Connect(); err != nil {
			log.Printf("Failed to connect to %s: %v", db.name, err)
		} else {
			log.Printf("Successfully connected to %s", db.name)
		}
	}
}

// DisconnectAll disconnects from all databases
func (cm *ConnectorManager) DisconnectAll() {
	databases := []struct {
		name      string
		connector DatabaseConnector
	}{
		{"MySQL", cm.MySQL},
		{"PostgreSQL", cm.PostgreSQL},
		{"MongoDB", cm.MongoDB},
		{"Redis", cm.Redis},
		{"MSSQL", cm.MSSQL},
	}

	for _, db := range databases {
		if err := db.connector.Disconnect(); err != nil {
			log.Printf("Failed to disconnect from %s: %v", db.name, err)
		} else {
			log.Printf("Successfully disconnected from %s", db.name)
		}
	}
}

// GetAllConnectors returns all connectors
func (cm *ConnectorManager) GetAllConnectors() map[string]DatabaseConnector {
	return map[string]DatabaseConnector{
		"MySQL":      cm.MySQL,
		"PostgreSQL": cm.PostgreSQL,
		"MongoDB":    cm.MongoDB,
		"Redis":      cm.Redis,
		"MSSQL":      cm.MSSQL,
	}
}

// UpdateConfigurations updates database configurations dynamically
func (cm *ConnectorManager) UpdateConfigurations(updates config.DynamicDatabaseConfig) error {
	var errors []string

	// Update MySQL configuration if provided
	if updates.MySQL != nil {
		if err := cm.MySQL.UpdateConfig(updates.MySQL); err != nil {
			errors = append(errors, fmt.Sprintf("MySQL: %v", err))
			log.Printf("Failed to update MySQL configuration: %v", err)
		} else {
			log.Printf("MySQL configuration updated successfully")
		}
	}

	// Update PostgreSQL configuration if provided
	if updates.PostgreSQL != nil {
		if err := cm.PostgreSQL.UpdateConfig(updates.PostgreSQL); err != nil {
			errors = append(errors, fmt.Sprintf("PostgreSQL: %v", err))
			log.Printf("Failed to update PostgreSQL configuration: %v", err)
		} else {
			log.Printf("PostgreSQL configuration updated successfully")
		}
	}

	// Update MongoDB configuration if provided
	if updates.MongoDB != nil {
		if err := cm.MongoDB.UpdateConfig(updates.MongoDB); err != nil {
			errors = append(errors, fmt.Sprintf("MongoDB: %v", err))
			log.Printf("Failed to update MongoDB configuration: %v", err)
		} else {
			log.Printf("MongoDB configuration updated successfully")
		}
	}

	// Update Redis configuration if provided
	if updates.Redis != nil {
		if err := cm.Redis.UpdateConfig(updates.Redis); err != nil {
			errors = append(errors, fmt.Sprintf("Redis: %v", err))
			log.Printf("Failed to update Redis configuration: %v", err)
		} else {
			log.Printf("Redis configuration updated successfully")
		}
	}

	// Update MSSQL configuration if provided
	if updates.MSSQL != nil {
		if err := cm.MSSQL.UpdateConfig(updates.MSSQL); err != nil {
			errors = append(errors, fmt.Sprintf("MSSQL: %v", err))
			log.Printf("Failed to update MSSQL configuration: %v", err)
		} else {
			log.Printf("MSSQL configuration updated successfully")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to update some configurations: %v", errors)
	}

	return nil
}

// GetConfigurations returns current configurations for all databases
func (cm *ConnectorManager) GetConfigurations() config.DynamicDatabaseConfig {
	return config.DynamicDatabaseConfig{
		MySQL:      cm.MySQL.config,
		PostgreSQL: cm.PostgreSQL.config,
		MongoDB:    cm.MongoDB.config,
		Redis:      cm.Redis.config,
		MSSQL:      cm.MSSQL.config,
	}
}

package stresstester

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"tcp-stress-test/connectors"
)

// TestConfig holds configuration for stress testing
type TestConfig struct {
	Duration         time.Duration // How long to run the test
	Concurrency      int           // Number of concurrent operations
	QueryType        string        // Query pattern: "simple", "complex", "heavy_read", "heavy_write", "mixed", "transaction", "bulk", "analytics"
	DelayBetween     time.Duration // Delay between operations
	OperationsLimit  int           // Maximum number of operations (-1 for unlimited)
	MixedRatio       MixedRatio    // Ratio for mixed workloads
	BurstMode        bool          // Enable burst mode for intensive testing
	RandomizeQueries bool          // Randomize query selection within type
}

// MixedRatio defines ratios for mixed workload testing
type MixedRatio struct {
	ReadPercent        int // Percentage of read operations (0-100)
	WritePercent       int // Percentage of write operations (0-100)
	TransactionPercent int // Percentage of transaction operations (0-100)
}

// TestResult holds the results of a stress test
type TestResult struct {
	DatabaseName     string
	StartTime        time.Time
	EndTime          time.Time
	TotalDuration    time.Duration
	TotalOperations  int64
	SuccessfulOps    int64
	FailedOps        int64
	OperationsPerSec float64
	AvgLatency       time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	Percentile95     time.Duration
	Percentile99     time.Duration
	ErrorRate        float64
	Errors           []string
}

// OperationResult holds the result of a single operation
type OperationResult struct {
	Success   bool
	Latency   time.Duration
	Error     string
	Timestamp time.Time
}

// StressTester manages stress testing operations
type StressTester struct {
	connectors map[string]connectors.DatabaseConnector
	config     *TestConfig
	results    map[string]*TestResult
	mutex      sync.RWMutex
}

// NewStressTester creates a new stress tester
func NewStressTester(conns map[string]connectors.DatabaseConnector, config *TestConfig) *StressTester {
	return &StressTester{
		connectors: conns,
		config:     config,
		results:    make(map[string]*TestResult),
	}
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		Duration:         30 * time.Second,
		Concurrency:      10,
		QueryType:        "simple",
		DelayBetween:     100 * time.Millisecond,
		OperationsLimit:  -1,
		MixedRatio:       MixedRatio{ReadPercent: 70, WritePercent: 25, TransactionPercent: 5},
		BurstMode:        false,
		RandomizeQueries: true,
	}
}

// DefaultMixedWorkloadConfig returns a configuration optimized for mixed workloads
func DefaultMixedWorkloadConfig() *TestConfig {
	return &TestConfig{
		Duration:         60 * time.Second,
		Concurrency:      20,
		QueryType:        "mixed",
		DelayBetween:     50 * time.Millisecond,
		OperationsLimit:  -1,
		MixedRatio:       MixedRatio{ReadPercent: 60, WritePercent: 30, TransactionPercent: 10},
		BurstMode:        true,
		RandomizeQueries: true,
	}
}

// DefaultHeavyLoadConfig returns a configuration for heavy load testing
func DefaultHeavyLoadConfig() *TestConfig {
	return &TestConfig{
		Duration:         120 * time.Second,
		Concurrency:      50,
		QueryType:        "heavy_read",
		DelayBetween:     10 * time.Millisecond,
		OperationsLimit:  -1,
		MixedRatio:       MixedRatio{ReadPercent: 80, WritePercent: 15, TransactionPercent: 5},
		BurstMode:        true,
		RandomizeQueries: true,
	}
}

// RunStressTest runs stress tests on all connected databases
func (st *StressTester) RunStressTest() map[string]*TestResult {
	var wg sync.WaitGroup

	log.Printf("Starting stress test with config: Duration=%v, Concurrency=%d, QueryType=%s",
		st.config.Duration, st.config.Concurrency, st.config.QueryType)

	for name, connector := range st.connectors {
		if !connector.IsConnected() {
			log.Printf("Skipping %s - not connected", name)
			continue
		}

		wg.Add(1)
		go func(dbName string, conn connectors.DatabaseConnector) {
			defer wg.Done()
			result := st.runSingleDatabaseTest(dbName, conn)

			st.mutex.Lock()
			st.results[dbName] = result
			st.mutex.Unlock()
		}(name, connector)
	}

	wg.Wait()

	log.Printf("Stress test completed for %d databases", len(st.results))
	return st.results
}

// runSingleDatabaseTest runs stress test on a single database
func (st *StressTester) runSingleDatabaseTest(dbName string, connector connectors.DatabaseConnector) *TestResult {
	log.Printf("Starting stress test for %s", dbName)

	// Initialize stress test tables
	if err := st.initializeStressTables(connector); err != nil {
		log.Printf("Warning: Could not initialize tables for %s: %v", dbName, err)
	}

	result := &TestResult{
		DatabaseName: dbName,
		StartTime:    time.Now(),
		MinLatency:   time.Hour, // Initialize with a large value
		Errors:       make([]string, 0),
	}

	ctx, cancel := context.WithTimeout(context.Background(), st.config.Duration)
	defer cancel()

	operationResults := make(chan OperationResult, st.config.Concurrency*2)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < st.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			st.worker(ctx, workerID, connector, operationResults)
		}(i)
	}

	// Start result collector
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		st.collectResults(operationResults, result)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(operationResults)

	// Wait for result collector to finish
	collectorWg.Wait()

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Calculate final metrics
	st.calculateFinalMetrics(result)

	log.Printf("Completed stress test for %s: %d operations, %.2f ops/sec, %.2f%% error rate",
		dbName, result.TotalOperations, result.OperationsPerSec, result.ErrorRate)

	return result
}

// executeHeavyReadOperation executes intensive read operations
func (st *StressTester) executeHeavyReadOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	// Database-specific heavy read queries to avoid syntax or missing-table errors.
	var operations []string

	switch connector.(type) {
	case *connectors.MySQLConnector:
		operations = []string{
			"SELECT u.*, COUNT(o.id) AS order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id ORDER BY order_count DESC LIMIT 1000",
			"SELECT * FROM orders WHERE order_date >= DATE_SUB(NOW(), INTERVAL 1 YEAR) ORDER BY total_amount DESC LIMIT 5000",
			"SELECT DATE(created_at) AS date, COUNT(*) AS count, AVG(total_amount) AS avg_amount FROM orders GROUP BY DATE(created_at) ORDER BY date DESC",
		}
	case *connectors.PostgreSQLConnector:
		operations = []string{
			"SELECT schemaname, tablename FROM pg_catalog.pg_tables LIMIT 1000",
			"SELECT relname, seq_scan, idx_scan FROM pg_stat_user_tables ORDER BY seq_scan DESC LIMIT 500",
			"SELECT datname, numbackends, xact_commit FROM pg_stat_database ORDER BY xact_commit DESC LIMIT 100",
		}
	case *connectors.MSSQLConnector:
		operations = []string{
			"SELECT TOP 1000 * FROM sys.tables",
			"SELECT TOP 1000 name, create_date, modify_date FROM sys.objects ORDER BY modify_date DESC",
			"SELECT TOP 1000 s.name AS schema_name, t.name AS table_name, p.rows FROM sys.tables t JOIN sys.schemas s ON t.schema_id = s.schema_id JOIN sys.partitions p ON t.object_id = p.object_id WHERE p.index_id IN (0,1) ORDER BY p.rows DESC",
		}
	case *connectors.MongoDBConnector:
		// Delegate to the connector’s complex query implementation for heavy MongoDB reads.
		return connector.ExecuteComplexQuery()
	default:
		operations = []string{
			"SELECT 1",
		}
	}

	var query string
	if st.config.RandomizeQueries && len(operations) > 1 {
		query = operations[st.getRandomIndex(len(operations))]
	} else {
		query = operations[0]
	}

	return connector.ExecuteQuery(query)
}

// executeHeavyWriteOperation executes intensive write operations
func (st *StressTester) executeHeavyWriteOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	var operations []string

	switch connector.(type) {
	case *connectors.MySQLConnector:
		operations = []string{
			// Bulk insert with multiple rows
			fmt.Sprintf(`INSERT INTO stress_test_%d (id, data, timestamp) VALUES
				(%d, 'data_%d', NOW()), (%d, 'data_%d', NOW()),
				(%d, 'data_%d', NOW()), (%d, 'data_%d', NOW()),
				(%d, 'data_%d', NOW()), (%d, 'data_%d', NOW()),
				(%d, 'data_%d', NOW()), (%d, 'data_%d', NOW()),
				(%d, 'data_%d', NOW()), (%d, 'data_%d', NOW())`,
				st.getRandomIndex(10),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000)),
			// Mass update with subquery
			fmt.Sprintf(`UPDATE stress_test_%d SET data = CONCAT('updated_', RAND() * 100000),
				timestamp = NOW() WHERE id IN (SELECT * FROM (SELECT id FROM stress_test_%d
				ORDER BY RAND() LIMIT 100) AS tmp)`, st.getRandomIndex(10), st.getRandomIndex(10)),
			// Create temporary table and populate
			fmt.Sprintf(`CREATE TEMPORARY TABLE IF NOT EXISTS temp_heavy_%d
				(id INT AUTO_INCREMENT PRIMARY KEY, data VARCHAR(255), ts TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
				st.getRandomIndex(10000)),
		}
	case *connectors.PostgreSQLConnector:
		operations = []string{
			// PostgreSQL bulk insert with generate_series
			fmt.Sprintf(`INSERT INTO stress_test_%d (id, data, timestamp)
				SELECT generate_series(%d, %d),
				'bulk_data_' || (random() * 100000)::int,
				NOW() + (random() * interval '30 days')`,
				st.getRandomIndex(10), st.getRandomIndex(1000), st.getRandomIndex(1000)+100),
			// Mass update with CTEs
			fmt.Sprintf(`WITH updates AS (
				SELECT id, 'heavy_update_' || (random() * 100000)::int as new_data
				FROM stress_test_%d TABLESAMPLE SYSTEM(10)
			) UPDATE stress_test_%d SET data = updates.new_data
			FROM updates WHERE stress_test_%d.id = updates.id`,
				st.getRandomIndex(10), st.getRandomIndex(10), st.getRandomIndex(10)),
			// Upsert with conflict handling
			fmt.Sprintf(`INSERT INTO stress_test_%d (id, data) VALUES (%d, 'conflict_%d')
				ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data || '_updated',
				update_count = stress_test_%d.update_count + 1`,
				st.getRandomIndex(10), st.getRandomIndex(10000), st.getRandomIndex(10000), st.getRandomIndex(10)),
		}
	case *connectors.MSSQLConnector:
		operations = []string{
			// MSSQL bulk insert using VALUES constructor
			fmt.Sprintf(`INSERT INTO stress_test_%d (id, data, timestamp) VALUES
				(%d, 'bulk_%d', GETDATE()), (%d, 'bulk_%d', GETDATE()),
				(%d, 'bulk_%d', GETDATE()), (%d, 'bulk_%d', GETDATE()),
				(%d, 'bulk_%d', GETDATE()), (%d, 'bulk_%d', GETDATE())`,
				st.getRandomIndex(10),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000),
				st.getRandomIndex(100000), st.getRandomIndex(100000)),
			// MERGE statement for upsert
			fmt.Sprintf(`MERGE stress_test_%d AS target
				USING (SELECT %d as id, 'merge_data_%d' as data) AS source
				ON target.id = source.id
				WHEN MATCHED THEN UPDATE SET data = source.data
				WHEN NOT MATCHED THEN INSERT (id, data) VALUES (source.id, source.data);`,
				st.getRandomIndex(10), st.getRandomIndex(10000), st.getRandomIndex(10000)),
			// Batch delete and recreate
			fmt.Sprintf(`DELETE TOP(1000) FROM stress_test_%d WHERE id %% 3 = 0`,
				st.getRandomIndex(10)),
		}
	default:
		// Fallback to simple operations
		operations = []string{
			fmt.Sprintf("INSERT INTO test_table VALUES (%d, 'data_%d')",
				st.getRandomIndex(100000), st.getRandomIndex(100000)),
		}
	}

	var query string
	if st.config.RandomizeQueries && len(operations) > 1 {
		query = operations[st.getRandomIndex(len(operations))]
	} else {
		query = operations[0]
	}

	return connector.ExecuteQuery(query)
}

// executeMixedOperation executes mixed read/write operations based on ratio
func (st *StressTester) executeMixedOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	rand := st.getRandomIndex(100)

	// In burst mode, execute multiple operations rapidly
	if st.config.BurstMode {
		// Execute 3-5 operations in quick succession
		numOps := 3 + st.getRandomIndex(3)
		var lastResult *connectors.QueryResult
		var lastErr error

		for i := 0; i < numOps; i++ {
			opRand := st.getRandomIndex(100)
			if opRand < st.config.MixedRatio.ReadPercent {
				lastResult, lastErr = st.executeHeavyReadOperation(connector)
			} else if opRand < st.config.MixedRatio.ReadPercent+st.config.MixedRatio.WritePercent {
				lastResult, lastErr = st.executeHeavyWriteOperation(connector)
			} else {
				lastResult, lastErr = st.executeTransactionOperation(connector)
			}

			if lastErr != nil {
				return lastResult, lastErr
			}
		}
		return lastResult, lastErr
	}

	// Normal mixed operation mode
	if rand < st.config.MixedRatio.ReadPercent {
		// Execute heavy read operation
		return st.executeHeavyReadOperation(connector)
	} else if rand < st.config.MixedRatio.ReadPercent+st.config.MixedRatio.WritePercent {
		// Execute heavy write operation
		return st.executeHeavyWriteOperation(connector)
	} else {
		// Execute transaction
		return st.executeTransactionOperation(connector)
	}
}

// executeTransactionOperation executes transaction-based operations
func (st *StressTester) executeTransactionOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	var transaction string

	switch connector.(type) {
	case *connectors.MySQLConnector:
		// MySQL complex transaction with multiple tables
		transaction = fmt.Sprintf(`
			START TRANSACTION;
			SET @test_id = %d;
			SET @test_val = RAND() * 10000;

			CREATE TEMPORARY TABLE IF NOT EXISTS trans_temp_%d (
				id INT PRIMARY KEY,
				value DECIMAL(10,2),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);

			INSERT INTO trans_temp_%d VALUES
				(@test_id, @test_val, NOW()),
				(@test_id + 1, @test_val * 1.1, NOW()),
				(@test_id + 2, @test_val * 1.2, NOW()),
				(@test_id + 3, @test_val * 1.3, NOW()),
				(@test_id + 4, @test_val * 1.4, NOW());

			UPDATE trans_temp_%d SET value = value * 1.05 WHERE id %% 2 = 0;

			DELETE FROM trans_temp_%d WHERE value < @test_val * 1.1;

			INSERT INTO trans_temp_%d
			SELECT id + 1000, value * 2, NOW() FROM trans_temp_%d WHERE id < @test_id + 2;

			COMMIT;`,
			st.getRandomIndex(100000), st.getRandomIndex(1000),
			st.getRandomIndex(1000), st.getRandomIndex(1000),
			st.getRandomIndex(1000), st.getRandomIndex(1000),
			st.getRandomIndex(1000))

	case *connectors.PostgreSQLConnector:
		// PostgreSQL transaction with CTEs and advanced features
		transaction = fmt.Sprintf(`
			BEGIN;

			WITH test_data AS (
				SELECT generate_series(1, 10) as id,
				       random() * 10000 as value,
				       NOW() + (random() * interval '30 days') as ts
			),
			inserted AS (
				INSERT INTO stress_trans_%d (id, value, timestamp)
				SELECT id + %d, value, ts FROM test_data
				RETURNING id, value
			),
			updated AS (
				UPDATE stress_trans_%d
				SET value = value * 1.1,
				    update_count = COALESCE(update_count, 0) + 1
				WHERE id IN (SELECT id FROM inserted WHERE value > 5000)
				RETURNING id, value
			)
			SELECT COUNT(*) as inserts,
			       (SELECT COUNT(*) FROM updated) as updates,
			       AVG(value) as avg_value
			FROM inserted;

			COMMIT;`,
			st.getRandomIndex(100), st.getRandomIndex(10000), st.getRandomIndex(100))

	case *connectors.MSSQLConnector:
		// MSSQL transaction with table variables and MERGE
		transaction = fmt.Sprintf(`
			BEGIN TRANSACTION;

			DECLARE @TestTable TABLE (
				id INT,
				value DECIMAL(10,2),
				status VARCHAR(20)
			);

			INSERT INTO @TestTable (id, value, status)
			SELECT TOP 10
				ROW_NUMBER() OVER (ORDER BY (SELECT NULL)) + %d,
				RAND(CHECKSUM(NEWID())) * 10000,
				CASE WHEN RAND() > 0.5 THEN 'active' ELSE 'pending' END
			FROM sys.objects;

			MERGE stress_trans_%d AS target
			USING @TestTable AS source ON target.id = source.id
			WHEN MATCHED AND source.value > target.value THEN
				UPDATE SET value = source.value, status = source.status
			WHEN NOT MATCHED THEN
				INSERT (id, value, status) VALUES (source.id, source.value, source.status);

			DELETE FROM stress_trans_%d
			WHERE id IN (SELECT id FROM @TestTable WHERE status = 'pending' AND value < 5000);

			COMMIT TRANSACTION;`,
			st.getRandomIndex(100000), st.getRandomIndex(100), st.getRandomIndex(100))

	default:
		// Fallback simple transaction
		transaction = "SELECT 1"
	}

	return connector.ExecuteQuery(transaction)
}

// executeBulkOperation executes bulk insert/update operations
func (st *StressTester) executeBulkOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	var bulkOp string
	batchSize := 50 // Increase batch size for heavy load

	switch connector.(type) {
	case *connectors.MySQLConnector:
		// MySQL bulk insert with ON DUPLICATE KEY UPDATE
		values := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			values[i] = fmt.Sprintf("(%d, 'bulk_%d', %.2f, NOW())",
				st.getRandomIndex(100000), st.getRandomIndex(100000), rand.Float64()*10000)
		}
		bulkOp = fmt.Sprintf(`INSERT INTO bulk_test_%d (id, data, value, timestamp) VALUES %s
			ON DUPLICATE KEY UPDATE
			value = VALUES(value) * 1.1,
			update_count = COALESCE(update_count, 0) + 1,
			timestamp = NOW()`,
			st.getRandomIndex(10), strings.Join(values, ", "))

	case *connectors.PostgreSQLConnector:
		// PostgreSQL bulk operations with COPY-like syntax
		bulkOp = fmt.Sprintf(`INSERT INTO bulk_test_%d (id, data, value, timestamp)
			SELECT
				generate_series(%d, %d),
				'bulk_' || (random() * 100000)::int,
				random() * 10000,
				NOW() + (random() * interval '30 days')
			ON CONFLICT (id) DO UPDATE SET
				value = EXCLUDED.value * 1.1,
				update_count = COALESCE(bulk_test_%d.update_count, 0) + 1`,
			st.getRandomIndex(10),
			st.getRandomIndex(10000), st.getRandomIndex(10000)+batchSize,
			st.getRandomIndex(10))

	case *connectors.MSSQLConnector:
		// MSSQL bulk operations with table-valued constructor
		values := make([]string, batchSize)
		for i := 0; i < batchSize; i++ {
			values[i] = fmt.Sprintf("(%d, 'bulk_%d', %.2f, GETDATE())",
				st.getRandomIndex(100000), st.getRandomIndex(100000), rand.Float64()*10000)
		}
		bulkOp = fmt.Sprintf(`
			MERGE bulk_test_%d AS target
			USING (VALUES %s) AS source (id, data, value, timestamp)
			ON target.id = source.id
			WHEN MATCHED THEN UPDATE SET
				data = source.data,
				value = source.value * 1.1,
				timestamp = source.timestamp
			WHEN NOT MATCHED THEN INSERT (id, data, value, timestamp)
				VALUES (source.id, source.data, source.value, source.timestamp);`,
			st.getRandomIndex(10), strings.Join(values, ", "))

	default:
		// Fallback simple bulk insert
		bulkOp = fmt.Sprintf("INSERT INTO test_table VALUES (%d, 'data_%d')",
			st.getRandomIndex(100000), st.getRandomIndex(100000))
	}

	return connector.ExecuteQuery(bulkOp)
}

// executeAnalyticsOperation executes analytics-heavy operations
func (st *StressTester) executeAnalyticsOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	var analytics string

	switch connector.(type) {
	case *connectors.MySQLConnector:
		analytics = `SELECT
			COUNT(*) as total_rows,
			AVG(CHAR_LENGTH(table_name)) as avg_name_length,
			COUNT(DISTINCT table_schema) as schemas,
			SUM(CASE WHEN engine = 'InnoDB' THEN 1 ELSE 0 END) as innodb_tables,
			MAX(create_time) as newest_table
		FROM information_schema.tables
		WHERE table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		GROUP BY table_type
		ORDER BY total_rows DESC`

	case *connectors.PostgreSQLConnector:
		analytics = `SELECT
			schemaname,
			COUNT(*) as table_count,
			SUM(n_live_tup) as total_rows,
			SUM(n_dead_tup) as dead_rows,
			AVG(n_mod_since_analyze) as avg_mods,
			MAX(last_vacuum) as last_vacuum_time,
			MAX(last_analyze) as last_analyze_time
		FROM pg_stat_user_tables
		GROUP BY schemaname
		ORDER BY total_rows DESC`

	case *connectors.MSSQLConnector:
		analytics = `SELECT
			s.name as schema_name,
			COUNT(*) as table_count,
			SUM(p.rows) as total_rows,
			AVG(p.rows) as avg_rows,
			COUNT(DISTINCT i.type_desc) as index_types,
			SUM(CASE WHEN i.is_primary_key = 1 THEN 1 ELSE 0 END) as pk_count
		FROM sys.tables t
		JOIN sys.schemas s ON t.schema_id = s.schema_id
		JOIN sys.partitions p ON t.object_id = p.object_id
		LEFT JOIN sys.indexes i ON t.object_id = i.object_id
		WHERE p.index_id IN (0, 1)
		GROUP BY s.name
		ORDER BY total_rows DESC`

	default:
		analytics = "SELECT COUNT(*) FROM test_table"
	}

	return connector.ExecuteQuery(analytics)
}

// initializeStressTables creates necessary tables for stress testing if they don't exist
func (st *StressTester) initializeStressTables(connector connectors.DatabaseConnector) error {
	var initQueries []string

	switch connector.(type) {
	case *connectors.MySQLConnector:
		initQueries = []string{
			`CREATE TABLE IF NOT EXISTS stress_test_0 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_1 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_2 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_3 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_4 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_5 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_6 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_7 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_8 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_9 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_0 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_1 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_2 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_3 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_4 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_5 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_6 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_7 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_8 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_9 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
		}

	case *connectors.PostgreSQLConnector:
		initQueries = []string{
			`CREATE TABLE IF NOT EXISTS stress_test_0 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_1 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_2 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_3 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_4 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_5 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_6 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_7 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_8 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_test_9 (id INT PRIMARY KEY, data VARCHAR(255), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
			`CREATE TABLE IF NOT EXISTS stress_trans_0 (id INT PRIMARY KEY, value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS stress_trans_1 (id INT PRIMARY KEY, value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS stress_trans_99 (id INT PRIMARY KEY, value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_0 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_1 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_2 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_3 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_4 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_5 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_6 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_7 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_8 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
			`CREATE TABLE IF NOT EXISTS bulk_test_9 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP, update_count INT DEFAULT 0)`,
		}

	case *connectors.MSSQLConnector:
		initQueries = []string{
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_0' AND xtype='U') CREATE TABLE stress_test_0 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_1' AND xtype='U') CREATE TABLE stress_test_1 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_2' AND xtype='U') CREATE TABLE stress_test_2 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_3' AND xtype='U') CREATE TABLE stress_test_3 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_4' AND xtype='U') CREATE TABLE stress_test_4 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_5' AND xtype='U') CREATE TABLE stress_test_5 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_6' AND xtype='U') CREATE TABLE stress_test_6 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_7' AND xtype='U') CREATE TABLE stress_test_7 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_8' AND xtype='U') CREATE TABLE stress_test_8 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_test_9' AND xtype='U') CREATE TABLE stress_test_9 (id INT PRIMARY KEY, data VARCHAR(255), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_trans_0' AND xtype='U') CREATE TABLE stress_trans_0 (id INT PRIMARY KEY, value DECIMAL(10,2), status VARCHAR(20))`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='stress_trans_99' AND xtype='U') CREATE TABLE stress_trans_99 (id INT PRIMARY KEY, value DECIMAL(10,2), status VARCHAR(20))`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_0' AND xtype='U') CREATE TABLE bulk_test_0 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_1' AND xtype='U') CREATE TABLE bulk_test_1 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_2' AND xtype='U') CREATE TABLE bulk_test_2 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_3' AND xtype='U') CREATE TABLE bulk_test_3 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_4' AND xtype='U') CREATE TABLE bulk_test_4 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_5' AND xtype='U') CREATE TABLE bulk_test_5 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_6' AND xtype='U') CREATE TABLE bulk_test_6 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_7' AND xtype='U') CREATE TABLE bulk_test_7 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_8' AND xtype='U') CREATE TABLE bulk_test_8 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
			`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='bulk_test_9' AND xtype='U') CREATE TABLE bulk_test_9 (id INT PRIMARY KEY, data VARCHAR(255), value DECIMAL(10,2), timestamp DATETIME DEFAULT GETDATE())`,
		}

	default:
		// No initialization needed for other connectors
		return nil
	}

	// Execute initialization queries
	for _, query := range initQueries {
		_, err := connector.ExecuteQuery(query)
		if err != nil {
			// Log but don't fail - tables might already exist
			log.Printf("Warning: Could not create table: %v", err)
		}
	}

	return nil
}

// executeConcurrentOperation executes operations designed for concurrency testing
func (st *StressTester) executeConcurrentOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	counterID := st.getRandomIndex(10) + 1
	concurrent := fmt.Sprintf(`
		UPDATE counters SET value = value + 1, updated_at = NOW() WHERE id = %d;
		SELECT value FROM counters WHERE id = %d;
	`, counterID, counterID)

	return connector.ExecuteQuery(concurrent)
}

// getRandomIndex returns a random index for various operations
func (st *StressTester) getRandomIndex(max int) int {
	return int(time.Now().UnixNano() % int64(max))
}

// worker performs database operations
func (st *StressTester) worker(ctx context.Context, workerID int, connector connectors.DatabaseConnector, results chan<- OperationResult) {
	operationCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check operation limit
			if st.config.OperationsLimit > 0 && operationCount >= st.config.OperationsLimit {
				return
			}

			// In burst mode, execute multiple operations rapidly
			if st.config.BurstMode {
				// Execute 5-10 operations in rapid succession without delay
				burstSize := 5 + st.getRandomIndex(6)
				for i := 0; i < burstSize; i++ {
					select {
					case <-ctx.Done():
						return
					default:
						opResult := st.performOperation(connector)
						select {
						case results <- opResult:
							operationCount++
						case <-ctx.Done():
							return
						}
					}
				}
				// Small delay after burst to prevent overwhelming the database
				if st.config.DelayBetween > 0 {
					select {
					case <-time.After(st.config.DelayBetween / 10): // Reduced delay in burst mode
					case <-ctx.Done():
						return
					}
				}
			} else {
				// Normal operation mode
				opResult := st.performOperation(connector)

				select {
				case results <- opResult:
					operationCount++
				case <-ctx.Done():
					return
				}

				// Add delay between operations if configured
				if st.config.DelayBetween > 0 {
					select {
					case <-time.After(st.config.DelayBetween):
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

// performOperation performs a single database operation
func (st *StressTester) performOperation(connector connectors.DatabaseConnector) OperationResult {
	start := time.Now()

	var queryResult *connectors.QueryResult
	var err error

	// Execute query based on type and workload pattern
	switch st.config.QueryType {
	case "simple":
		queryResult, err = connector.ExecuteQuery("SELECT 1")
	case "complex":
		queryResult, err = connector.ExecuteComplexQuery()
	case "heavy_read":
		queryResult, err = st.executeHeavyReadOperation(connector)
	case "heavy_write":
		queryResult, err = st.executeHeavyWriteOperation(connector)
	case "mixed":
		queryResult, err = st.executeMixedOperation(connector)
	case "transaction":
		queryResult, err = st.executeTransactionOperation(connector)
	case "bulk":
		queryResult, err = st.executeBulkOperation(connector)
	case "analytics":
		queryResult, err = st.executeAnalyticsOperation(connector)
	case "concurrent":
		queryResult, err = st.executeConcurrentOperation(connector)
	default:
		queryResult, err = connector.ExecuteQuery("SELECT 1")
	}

	latency := time.Since(start)

	result := OperationResult{
		Latency:   latency,
		Timestamp: time.Now(),
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else if queryResult != nil && !queryResult.Success {
		result.Success = false
		result.Error = queryResult.Message
	} else {
		result.Success = true
	}

	return result
}

// collectResults collects operation results and updates test result
func (st *StressTester) collectResults(results <-chan OperationResult, testResult *TestResult) {
	latencies := make([]time.Duration, 0, 1000)

	for opResult := range results {
		testResult.TotalOperations++
		latencies = append(latencies, opResult.Latency)

		if opResult.Success {
			testResult.SuccessfulOps++
		} else {
			testResult.FailedOps++
			if len(testResult.Errors) < 100 { // Limit error storage
				testResult.Errors = append(testResult.Errors, opResult.Error)
			}
		}

		// Update latency metrics
		if opResult.Latency < testResult.MinLatency {
			testResult.MinLatency = opResult.Latency
		}
		if opResult.Latency > testResult.MaxLatency {
			testResult.MaxLatency = opResult.Latency
		}
	}

	// Calculate percentiles
	if len(latencies) > 0 {
		st.calculatePercentiles(latencies, testResult)
	}
}

// calculatePercentiles calculates latency percentiles
func (st *StressTester) calculatePercentiles(latencies []time.Duration, result *TestResult) {
	// Simple bubble sort for percentile calculation
	for i := 0; i < len(latencies); i++ {
		for j := 0; j < len(latencies)-i-1; j++ {
			if latencies[j] > latencies[j+1] {
				latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
			}
		}
	}

	length := len(latencies)
	if length > 0 {
		p95Index := int(float64(length) * 0.95)
		p99Index := int(float64(length) * 0.99)

		if p95Index >= length {
			p95Index = length - 1
		}
		if p99Index >= length {
			p99Index = length - 1
		}

		result.Percentile95 = latencies[p95Index]
		result.Percentile99 = latencies[p99Index]

		// Calculate average latency
		var total time.Duration
		for _, latency := range latencies {
			total += latency
		}
		result.AvgLatency = total / time.Duration(length)
	}
}

// calculateFinalMetrics calculates final test metrics
func (st *StressTester) calculateFinalMetrics(result *TestResult) {
	if result.TotalOperations > 0 {
		result.ErrorRate = (float64(result.FailedOps) / float64(result.TotalOperations)) * 100
		result.OperationsPerSec = float64(result.TotalOperations) / result.TotalDuration.Seconds()
	}

	// Handle edge case where no operations failed
	if result.MinLatency == time.Hour {
		result.MinLatency = 0
	}
}

// GetResults returns current test results
func (st *StressTester) GetResults() map[string]*TestResult {
	st.mutex.RLock()
	defer st.mutex.RUnlock()

	// Create a copy to avoid race conditions
	results := make(map[string]*TestResult)
	for k, v := range st.results {
		results[k] = v
	}
	return results
}

// RunQuickConnectivityTest runs a quick connectivity test on all databases
func (st *StressTester) RunQuickConnectivityTest() map[string]bool {
	results := make(map[string]bool)

	for name, connector := range st.connectors {
		err := connector.TestConnection()
		results[name] = err == nil

		if err != nil {
			log.Printf("Connectivity test failed for %s: %v", name, err)
		} else {
			log.Printf("Connectivity test passed for %s", name)
		}
	}

	return results
}

// RunCustomTest runs a custom test with specific parameters
func (st *StressTester) RunCustomTest(dbName string, customConfig *TestConfig) (*TestResult, error) {
	connector, exists := st.connectors[dbName]
	if !exists {
		return nil, fmt.Errorf("database %s not found", dbName)
	}

	if !connector.IsConnected() {
		return nil, fmt.Errorf("database %s is not connected", dbName)
	}

	// Temporarily change config
	originalConfig := st.config
	st.config = customConfig
	defer func() {
		st.config = originalConfig
	}()

	log.Printf("Running custom test for %s with config: Duration=%v, Concurrency=%d",
		dbName, customConfig.Duration, customConfig.Concurrency)

	result := st.runSingleDatabaseTest(dbName, connector)
	return result, nil
}

// GetConnectionStatus returns the connection status of all databases
func (st *StressTester) GetConnectionStatus() map[string]string {
	status := make(map[string]string)

	for name, connector := range st.connectors {
		if connector.IsConnected() {
			status[name] = "Connected"
		} else {
			status[name] = "Disconnected"
		}
	}

	return status
}

// BenchmarkQuery runs a benchmark on a specific query
func (st *StressTester) BenchmarkQuery(dbName string, customQuery string, iterations int) (*TestResult, error) {
	connector, exists := st.connectors[dbName]
	if !exists {
		return nil, fmt.Errorf("database %s not found", dbName)
	}

	if !connector.IsConnected() {
		return nil, fmt.Errorf("database %s is not connected", dbName)
	}

	log.Printf("Running benchmark for %s with %d iterations", dbName, iterations)

	result := &TestResult{
		DatabaseName: dbName,
		StartTime:    time.Now(),
		MinLatency:   time.Hour,
		Errors:       make([]string, 0),
	}

	latencies := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		queryResult, err := connector.ExecuteQuery(customQuery)
		latency := time.Since(start)

		result.TotalOperations++
		latencies = append(latencies, latency)

		if err != nil {
			result.FailedOps++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors, err.Error())
			}
		} else if queryResult != nil && !queryResult.Success {
			result.FailedOps++
			if len(result.Errors) < 10 {
				result.Errors = append(result.Errors, queryResult.Message)
			}
		} else {
			result.SuccessfulOps++
		}

		// Update latency metrics
		if latency < result.MinLatency {
			result.MinLatency = latency
		}
		if latency > result.MaxLatency {
			result.MaxLatency = latency
		}
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Calculate percentiles and final metrics
	if len(latencies) > 0 {
		st.calculatePercentiles(latencies, result)
	}
	st.calculateFinalMetrics(result)

	log.Printf("Benchmark completed for %s: %d operations, %.2f ops/sec, %.2f%% error rate",
		dbName, result.TotalOperations, result.OperationsPerSec, result.ErrorRate)

	return result, nil
}

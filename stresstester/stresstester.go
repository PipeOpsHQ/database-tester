package stresstester

import (
	"context"
	"fmt"
	"log"
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
	operations := []string{
		fmt.Sprintf("INSERT INTO users (username, email, age, created_at) VALUES ('stress_user_%d', 'stress_%d@test.com', %d, NOW())",
			st.getRandomIndex(100000), st.getRandomIndex(100000), 20+st.getRandomIndex(50)),
		fmt.Sprintf("UPDATE users SET last_login = NOW(), login_count = login_count + 1 WHERE id = %d", st.getRandomIndex(1000)+1),
		fmt.Sprintf("DELETE FROM temp_data WHERE created_at < DATE_SUB(NOW(), INTERVAL %d HOUR)", st.getRandomIndex(24)+1),
	}

	if st.config.RandomizeQueries {
		operation := operations[st.getRandomIndex(len(operations))]
		return connector.ExecuteQuery(operation)
	}
	return connector.ExecuteQuery(operations[0])
}

// executeMixedOperation executes mixed read/write operations based on ratio
func (st *StressTester) executeMixedOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	// Generate random number to determine operation type based on ratio
	rand := st.getRandomIndex(100)

	if rand < st.config.MixedRatio.ReadPercent {
		return st.executeHeavyReadOperation(connector)
	} else if rand < st.config.MixedRatio.ReadPercent+st.config.MixedRatio.WritePercent {
		return st.executeHeavyWriteOperation(connector)
	} else {
		return st.executeTransactionOperation(connector)
	}
}

// executeTransactionOperation executes transaction-based operations
func (st *StressTester) executeTransactionOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	// Simulate complex transaction
	userID := st.getRandomIndex(1000) + 1
	amount := float64(st.getRandomIndex(50000)) / 100.0

	transaction := fmt.Sprintf(`
		START TRANSACTION;
		INSERT INTO orders (user_id, total_amount, status, order_date) VALUES (%d, %.2f, 'pending', NOW());
		UPDATE users SET total_spent = total_spent + %.2f WHERE id = %d;
		INSERT INTO audit_log (user_id, action, amount, timestamp) VALUES (%d, 'order_created', %.2f, NOW());
		COMMIT;
	`, userID, amount, amount, userID, userID, amount)

	return connector.ExecuteQuery(transaction)
}

// executeBulkOperation executes bulk insert/update operations
func (st *StressTester) executeBulkOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	// Generate bulk insert
	values := make([]string, 10)
	for i := 0; i < 10; i++ {
		values[i] = fmt.Sprintf("('bulk_user_%d_%d', 'bulk_%d_%d@test.com', %d, NOW())",
			st.getRandomIndex(10000), i, st.getRandomIndex(10000), i, 20+st.getRandomIndex(50))
	}

	bulkInsert := fmt.Sprintf("INSERT INTO users (username, email, age, created_at) VALUES %s",
		fmt.Sprintf("%s", values[0]))
	for i := 1; i < len(values); i++ {
		bulkInsert += ", " + values[i]
	}

	return connector.ExecuteQuery(bulkInsert)
}

// executeAnalyticsOperation executes analytics-heavy operations
func (st *StressTester) executeAnalyticsOperation(connector connectors.DatabaseConnector) (*connectors.QueryResult, error) {
	analytics := []string{
		`SELECT
			DATE(o.order_date) as date,
			COUNT(*) as orders,
			SUM(o.total_amount) as revenue,
			AVG(o.total_amount) as avg_order,
			COUNT(DISTINCT o.user_id) as customers
		FROM orders o
		WHERE o.order_date >= DATE_SUB(NOW(), INTERVAL 90 DAY)
		GROUP BY DATE(o.order_date)
		ORDER BY date DESC`,
		`SELECT
			u.age,
			COUNT(*) as user_count,
			AVG(u.total_spent) as avg_spent,
			SUM(o.total_amount) as total_revenue
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		GROUP BY u.age
		HAVING user_count > 5
		ORDER BY total_revenue DESC`,
	}

	if st.config.RandomizeQueries {
		query := analytics[st.getRandomIndex(len(analytics))]
		return connector.ExecuteQuery(query)
	}
	return connector.ExecuteQuery(analytics[0])
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

			// Perform operation
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

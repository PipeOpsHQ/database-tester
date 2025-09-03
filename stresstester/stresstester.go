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
	Duration        time.Duration // How long to run the test
	Concurrency     int           // Number of concurrent operations
	QueryType       string        // "simple" or "complex"
	DelayBetween    time.Duration // Delay between operations
	OperationsLimit int           // Maximum number of operations (-1 for unlimited)
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
		Duration:        30 * time.Second,
		Concurrency:     10,
		QueryType:       "simple",
		DelayBetween:    100 * time.Millisecond,
		OperationsLimit: -1,
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

	// Execute query based on type
	if st.config.QueryType == "complex" {
		queryResult, err = connector.ExecuteComplexQuery()
	} else {
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

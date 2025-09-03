package queries

import (
	"fmt"
	"math/rand"
	"time"
)

// QueryType represents different types of stress test queries
type QueryType string

const (
	QueryTypeSimple    QueryType = "simple"
	QueryTypeComplex   QueryType = "complex"
	QueryTypeRead      QueryType = "read"
	QueryTypeWrite     QueryType = "write"
	QueryTypeMixed     QueryType = "mixed"
	QueryTypeAnalytics QueryType = "analytics"
	QueryTypeOLTP      QueryType = "oltp"
	QueryTypeJoin      QueryType = "join"
	QueryTypeAggregate QueryType = "aggregate"
	QueryTypeRecursive QueryType = "recursive"
)

// DatabaseType represents supported database types
type DatabaseType string

const (
	DatabaseMySQL      DatabaseType = "mysql"
	DatabasePostgreSQL DatabaseType = "postgresql"
	DatabaseRedis      DatabaseType = "redis"
	DatabaseMSSQL      DatabaseType = "mssql"
	DatabaseMongoDB    DatabaseType = "mongodb"
)

// Query represents a database query with metadata
type Query struct {
	Name        string
	Description string
	SQL         string
	Type        QueryType
	Database    DatabaseType
	Complexity  int // 1-10 scale
	Expected    string
}

// QueryLibrary holds all available queries organized by database and type
type QueryLibrary struct {
	queries map[DatabaseType]map[QueryType][]Query
}

// NewQueryLibrary creates and initializes a new query library
func NewQueryLibrary() *QueryLibrary {
	ql := &QueryLibrary{
		queries: make(map[DatabaseType]map[QueryType][]Query),
	}
	ql.initializeQueries()
	return ql
}

// GetQuery returns a specific query by database type and query type
func (ql *QueryLibrary) GetQuery(dbType DatabaseType, queryType QueryType) (Query, error) {
	if dbQueries, exists := ql.queries[dbType]; exists {
		if queries, exists := dbQueries[queryType]; exists && len(queries) > 0 {
			// Return random query of the specified type
			return queries[rand.Intn(len(queries))], nil
		}
	}
	return Query{}, fmt.Errorf("no query found for database %s and type %s", dbType, queryType)
}

// GetAllQueries returns all queries for a specific database
func (ql *QueryLibrary) GetAllQueries(dbType DatabaseType) (map[QueryType][]Query, error) {
	if dbQueries, exists := ql.queries[dbType]; exists {
		return dbQueries, nil
	}
	return nil, fmt.Errorf("no queries found for database %s", dbType)
}

// GetQueryTypes returns available query types for a database
func (ql *QueryLibrary) GetQueryTypes(dbType DatabaseType) []QueryType {
	var types []QueryType
	if dbQueries, exists := ql.queries[dbType]; exists {
		for queryType := range dbQueries {
			types = append(types, queryType)
		}
	}
	return types
}

// initializeQueries populates the query library with predefined queries
func (ql *QueryLibrary) initializeQueries() {
	// Initialize maps
	for _, dbType := range []DatabaseType{DatabaseMySQL, DatabasePostgreSQL, DatabaseRedis, DatabaseMSSQL, DatabaseMongoDB} {
		ql.queries[dbType] = make(map[QueryType][]Query)
	}

	// MySQL Queries
	ql.initializeMySQLQueries()

	// PostgreSQL Queries
	ql.initializePostgreSQLQueries()

	// Redis Queries
	ql.initializeRedisQueries()

	// MSSQL Queries
	ql.initializeMSSQLQueries()

	// MongoDB Queries
	ql.initializeMongoDBQueries()
}

// initializeMySQLQueries adds MySQL-specific queries
func (ql *QueryLibrary) initializeMySQLQueries() {
	db := DatabaseMySQL

	// Simple queries
	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Basic Select",
		Description: "Simple connectivity test",
		SQL:         "SELECT 1 as test_value",
		Complexity:  1,
		Expected:    "Single row with value 1",
	})

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Current Time",
		Description: "Get current database time",
		SQL:         "SELECT NOW() as current_time, VERSION() as version",
		Complexity:  1,
		Expected:    "Current timestamp and MySQL version",
	})

	// Read queries
	ql.addQuery(db, QueryTypeRead, Query{
		Name:        "User Count",
		Description: "Count total users",
		SQL:         "SELECT COUNT(*) as user_count FROM users WHERE is_active = 1",
		Complexity:  2,
		Expected:    "Total count of active users",
	})

	ql.addQuery(db, QueryTypeRead, Query{
		Name:        "Recent Orders",
		Description: "Fetch recent orders with user info",
		SQL: `SELECT o.id, o.order_number, o.total_amount, u.username, o.created_at
			   FROM orders o
			   JOIN users u ON o.user_id = u.id
			   WHERE o.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
			   ORDER BY o.created_at DESC
			   LIMIT 100`,
		Complexity: 4,
		Expected:   "Recent orders with user details",
	})

	// Complex analytical queries
	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "Revenue Analysis",
		Description: "Monthly revenue analysis with growth metrics",
		SQL: `WITH monthly_revenue AS (
			SELECT
				DATE_FORMAT(order_date, '%Y-%m') as month,
				SUM(total_amount) as revenue,
				COUNT(*) as order_count,
				AVG(total_amount) as avg_order_value
			FROM orders
			WHERE status IN ('delivered', 'shipped')
			  AND order_date >= DATE_SUB(NOW(), INTERVAL 12 MONTH)
			GROUP BY DATE_FORMAT(order_date, '%Y-%m')
		),
		revenue_with_growth AS (
			SELECT
				month,
				revenue,
				order_count,
				avg_order_value,
				LAG(revenue) OVER (ORDER BY month) as prev_revenue,
				((revenue - LAG(revenue) OVER (ORDER BY month)) /
				 LAG(revenue) OVER (ORDER BY month)) * 100 as growth_rate
			FROM monthly_revenue
		)
		SELECT
			month,
			ROUND(revenue, 2) as revenue,
			order_count,
			ROUND(avg_order_value, 2) as avg_order_value,
			ROUND(COALESCE(growth_rate, 0), 2) as growth_percentage
		FROM revenue_with_growth
		ORDER BY month DESC`,
		Complexity: 8,
		Expected:   "Monthly revenue with growth analysis",
	})

	// Write queries
	ql.addQuery(db, QueryTypeWrite, Query{
		Name:        "Insert Test User",
		Description: "Insert a new test user",
		SQL: fmt.Sprintf(`INSERT INTO users (username, email, first_name, last_name, age, salary)
			   VALUES ('test_user_%d', 'test_%d@example.com', 'Test', 'User', %d, %.2f)`,
			rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), 30000+rand.Float64()*70000),
		Complexity: 3,
		Expected:   "New user inserted successfully",
	})

	// OLTP queries
	ql.addQuery(db, QueryTypeOLTP, Query{
		Name:        "Order Processing",
		Description: "Simulate order processing workflow",
		SQL: `UPDATE orders
			   SET status = 'processing', updated_at = NOW()
			   WHERE status = 'pending'
			   AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
			   LIMIT 10`,
		Complexity: 5,
		Expected:   "Order status updates",
	})

	// Join queries
	ql.addQuery(db, QueryTypeJoin, Query{
		Name:        "Customer Order Summary",
		Description: "Complex multi-table join with aggregations",
		SQL: `SELECT
			u.id,
			u.username,
			u.email,
			COUNT(DISTINCT o.id) as total_orders,
			COALESCE(SUM(o.total_amount), 0) as total_spent,
			COALESCE(AVG(o.total_amount), 0) as avg_order_value,
			COUNT(DISTINCT oi.id) as total_items,
			MAX(o.order_date) as last_order_date,
			CASE
				WHEN SUM(o.total_amount) > 1000 THEN 'Premium'
				WHEN SUM(o.total_amount) > 500 THEN 'Gold'
				WHEN SUM(o.total_amount) > 100 THEN 'Silver'
				ELSE 'Bronze'
			END as customer_tier
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id AND o.status != 'cancelled'
		LEFT JOIN order_items oi ON o.id = oi.order_id
		GROUP BY u.id, u.username, u.email
		HAVING COUNT(DISTINCT o.id) > 0
		ORDER BY total_spent DESC
		LIMIT 50`,
		Complexity: 7,
		Expected:   "Customer analytics with tier classification",
	})
}

// initializePostgreSQLQueries adds PostgreSQL-specific queries
func (ql *QueryLibrary) initializePostgreSQLQueries() {
	db := DatabasePostgreSQL

	// Simple queries
	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Basic Select",
		Description: "Simple connectivity test",
		SQL:         "SELECT 1 as test_value, current_timestamp as server_time",
		Complexity:  1,
		Expected:    "Single row with test value and timestamp",
	})

	// Advanced PostgreSQL features
	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "Time Series Analysis",
		Description: "Advanced time series analysis with window functions",
		SQL: `WITH daily_metrics AS (
			SELECT
				date_trunc('day', generate_series(
					current_date - interval '30 days',
					current_date,
					interval '1 day'
				)) as date,
				random() * 1000 as value,
				random() * 100 as secondary_metric
		),
		windowed_metrics AS (
			SELECT
				date,
				value,
				secondary_metric,
				AVG(value) OVER (ORDER BY date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) as moving_avg_7d,
				STDDEV(value) OVER (ORDER BY date ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) as moving_stddev,
				LAG(value, 1) OVER (ORDER BY date) as prev_value,
				LEAD(value, 1) OVER (ORDER BY date) as next_value,
				PERCENT_RANK() OVER (ORDER BY value) as percentile_rank
			FROM daily_metrics
		)
		SELECT
			date,
			ROUND(value::numeric, 2) as value,
			ROUND(moving_avg_7d::numeric, 2) as moving_avg_7d,
			ROUND(moving_stddev::numeric, 2) as volatility,
			CASE
				WHEN value > moving_avg_7d + moving_stddev THEN 'HIGH'
				WHEN value < moving_avg_7d - moving_stddev THEN 'LOW'
				ELSE 'NORMAL'
			END as trend_signal,
			ROUND((percentile_rank * 100)::numeric, 1) as percentile
		FROM windowed_metrics
		WHERE moving_avg_7d IS NOT NULL
		ORDER BY date DESC`,
		Complexity: 9,
		Expected:   "Time series analysis with statistical metrics",
	})

	// Recursive CTE
	ql.addQuery(db, QueryTypeRecursive, Query{
		Name:        "Fibonacci Sequence",
		Description: "Generate Fibonacci sequence using recursive CTE",
		SQL: `WITH RECURSIVE fibonacci(n, fib_n, fib_n_plus_1) AS (
			SELECT 1, 0::bigint, 1::bigint
			UNION ALL
			SELECT n + 1, fib_n_plus_1, fib_n + fib_n_plus_1
			FROM fibonacci
			WHERE n < 20
		)
		SELECT n as position, fib_n as fibonacci_number
		FROM fibonacci
		ORDER BY n`,
		Complexity: 6,
		Expected:   "Fibonacci sequence up to position 20",
	})

	// Array and JSON operations
	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "JSON and Array Processing",
		Description: "Complex JSON and array operations",
		SQL: `WITH sample_data AS (
			SELECT
				generate_series(1, 100) as id,
				jsonb_build_object(
					'name', 'User ' || generate_series(1, 100),
					'age', 18 + (random() * 50)::int,
					'tags', array['tag1', 'tag2', 'tag' || (random() * 10)::int],
					'metadata', jsonb_build_object(
						'created', now(),
						'score', random() * 100,
						'active', random() > 0.5
					)
				) as data
		)
		SELECT
			id,
			data->>'name' as name,
			(data->>'age')::int as age,
			array_length(string_to_array(data->>'tags', ','), 1) as tag_count,
			(data->'metadata'->>'score')::float as score,
			(data->'metadata'->>'active')::boolean as is_active
		FROM sample_data
		WHERE (data->>'age')::int > 25
		  AND (data->'metadata'->>'active')::boolean = true
		ORDER BY (data->'metadata'->>'score')::float DESC
		LIMIT 20`,
		Complexity: 7,
		Expected:   "JSON processing with filtering and sorting",
	})
}

// initializeRedisQueries adds Redis-specific operations
func (ql *QueryLibrary) initializeRedisQueries() {
	db := DatabaseRedis

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Basic GET",
		Description: "Simple key-value retrieval",
		SQL:         "GET test-key-" + fmt.Sprintf("%d", rand.Intn(1000)),
		Complexity:  1,
		Expected:    "Key value or nil",
	})

	ql.addQuery(db, QueryTypeWrite, Query{
		Name:        "SET with Expiry",
		Description: "Set key with expiration",
		SQL:         fmt.Sprintf("SETEX stress-key-%d 3600 'value-%d-%d'", rand.Intn(1000), rand.Intn(1000), time.Now().Unix()),
		Complexity:  2,
		Expected:    "OK response",
	})

	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Pipeline Operations",
		Description: "Multiple operations in pipeline",
		SQL:         "PIPELINE:SET|GET|INCR|EXPIRE", // Special format for Redis pipeline
		Complexity:  5,
		Expected:    "Multiple operation results",
	})
}

// initializeMSSQLQueries adds MSSQL-specific queries
func (ql *QueryLibrary) initializeMSSQLQueries() {
	db := DatabaseMSSQL

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Basic Select",
		Description: "Simple connectivity test",
		SQL:         "SELECT 1 as test_value, GETDATE() as server_time, @@VERSION as version",
		Complexity:  1,
		Expected:    "Test value with server info",
	})

	ql.addQuery(db, QueryTypeRecursive, Query{
		Name:        "Hierarchical Data",
		Description: "Recursive CTE for hierarchical data processing",
		SQL: `WITH NumberHierarchy AS (
			-- Anchor: Start with level 1
			SELECT 1 as Level, 1 as Number, 1 as ParentNumber
			UNION ALL
			-- Recursive: Generate next levels
			SELECT
				nh.Level + 1,
				nh.Number + nh.Level + 1,
				nh.Number
			FROM NumberHierarchy nh
			WHERE nh.Level < 5
		),
		HierarchyWithStats AS (
			SELECT
				Level,
				Number,
				ParentNumber,
				SUM(Number) OVER (PARTITION BY Level) as LevelSum,
				COUNT(*) OVER (PARTITION BY Level) as LevelCount,
				ROW_NUMBER() OVER (PARTITION BY Level ORDER BY Number) as RowInLevel
			FROM NumberHierarchy
		)
		SELECT
			Level,
			Number,
			ParentNumber,
			LevelSum,
			LevelCount,
			RowInLevel,
			CAST(REPLICATE('  ', Level - 1) + CAST(Number as VARCHAR(10)) as VARCHAR(50)) as HierarchyView
		FROM HierarchyWithStats
		ORDER BY Level, Number
		OPTION (MAXRECURSION 100)`,
		Complexity: 8,
		Expected:   "Hierarchical data with statistics",
	})

	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "Advanced Analytics",
		Description: "Complex analytical query with multiple CTEs",
		SQL: `WITH DateRange AS (
			SELECT CAST('2023-01-01' as DATE) as StartDate, CAST('2023-12-31' as DATE) as EndDate
		),
		CalendarDays AS (
			SELECT
				DATEADD(day, number, dr.StartDate) as CalendarDate
			FROM master.dbo.spt_values sv
			CROSS JOIN DateRange dr
			WHERE sv.type = 'P'
			  AND DATEADD(day, sv.number, dr.StartDate) <= dr.EndDate
		),
		DailySales AS (
			SELECT
				cd.CalendarDate,
				COALESCE(COUNT(o.id), 0) as OrderCount,
				COALESCE(SUM(o.total_amount), 0) as Revenue,
				COALESCE(AVG(o.total_amount), 0) as AvgOrderValue
			FROM CalendarDays cd
			LEFT JOIN orders o ON CAST(o.order_date as DATE) = cd.CalendarDate
			GROUP BY cd.CalendarDate
		),
		SalesWithTrends AS (
			SELECT
				CalendarDate,
				OrderCount,
				Revenue,
				AvgOrderValue,
				AVG(Revenue) OVER (ORDER BY CalendarDate ROWS BETWEEN 6 PRECEDING AND CURRENT ROW) as MovingAvg7Days,
				STDEV(Revenue) OVER (ORDER BY CalendarDate ROWS BETWEEN 29 PRECEDING AND CURRENT ROW) as Volatility30Days,
				LAG(Revenue, 7) OVER (ORDER BY CalendarDate) as RevenueLastWeek
			FROM DailySales
		)
		SELECT TOP 100
			CalendarDate,
			OrderCount,
			CAST(Revenue as DECIMAL(10,2)) as Revenue,
			CAST(AvgOrderValue as DECIMAL(10,2)) as AvgOrderValue,
			CAST(MovingAvg7Days as DECIMAL(10,2)) as MovingAvg7Days,
			CAST(Volatility30Days as DECIMAL(10,2)) as Volatility30Days,
			CASE
				WHEN RevenueLastWeek > 0 THEN
					CAST(((Revenue - RevenueLastWeek) / RevenueLastWeek * 100) as DECIMAL(10,2))
				ELSE 0
			END as WeekOverWeekGrowth
		FROM SalesWithTrends
		WHERE CalendarDate >= DATEADD(day, -100, GETDATE())
		ORDER BY CalendarDate DESC`,
		Complexity: 9,
		Expected:   "Advanced sales analytics with trends",
	})
}

// initializeMongoDBQueries adds MongoDB-specific operations
func (ql *QueryLibrary) initializeMongoDBQueries() {
	db := DatabaseMongoDB

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Find Documents",
		Description: "Simple document find operation",
		SQL:         `db.testcollection.find({}).limit(10)`,
		Complexity:  1,
		Expected:    "List of documents",
	})

	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Aggregation Pipeline",
		Description: "Complex aggregation with multiple stages",
		SQL: `db.testcollection.aggregate([
			{"$addFields": {
				"randomValue": {"$rand": {}},
				"category": {"$mod": [{"$toInt": "$_id"}, 5]}
			}},
			{"$group": {
				"_id": "$category",
				"count": {"$sum": 1},
				"avgValue": {"$avg": "$randomValue"},
				"maxValue": {"$max": "$randomValue"},
				"minValue": {"$min": "$randomValue"}
			}},
			{"$sort": {"_id": 1}},
			{"$project": {
				"category": "$_id",
				"count": 1,
				"avgValue": 1,
				"maxValue": 1,
				"minValue": 1,
				"range": {"$subtract": ["$maxValue", "$minValue"]},
				"_id": 0
			}}
		])`,
		Complexity: 7,
		Expected:   "Aggregated results with statistics",
	})
}

// addQuery adds a query to the library
func (ql *QueryLibrary) addQuery(dbType DatabaseType, queryType QueryType, query Query) {
	query.Type = queryType
	query.Database = dbType

	if ql.queries[dbType][queryType] == nil {
		ql.queries[dbType][queryType] = make([]Query, 0)
	}

	ql.queries[dbType][queryType] = append(ql.queries[dbType][queryType], query)
}

// GetRandomQuery returns a random query for any database and type
func (ql *QueryLibrary) GetRandomQuery() Query {
	// Get all databases
	databases := []DatabaseType{DatabaseMySQL, DatabasePostgreSQL, DatabaseRedis, DatabaseMSSQL}
	db := databases[rand.Intn(len(databases))]

	// Get available query types for this database
	queryTypes := ql.GetQueryTypes(db)
	if len(queryTypes) == 0 {
		return Query{}
	}

	queryType := queryTypes[rand.Intn(len(queryTypes))]
	query, _ := ql.GetQuery(db, queryType)
	return query
}

// GetQueryStats returns statistics about the query library
func (ql *QueryLibrary) GetQueryStats() map[string]interface{} {
	stats := make(map[string]interface{})
	totalQueries := 0

	for dbType, queryTypes := range ql.queries {
		dbStats := make(map[string]int)
		dbTotal := 0

		for queryType, queries := range queryTypes {
			count := len(queries)
			dbStats[string(queryType)] = count
			dbTotal += count
			totalQueries += count
		}

		dbStats["total"] = dbTotal
		stats[string(dbType)] = dbStats
	}

	stats["total_queries"] = totalQueries
	return stats
}

// ValidateQuery checks if a query is valid for stress testing
func (ql *QueryLibrary) ValidateQuery(query Query) []string {
	var issues []string

	if query.Name == "" {
		issues = append(issues, "Query name is required")
	}

	if query.SQL == "" {
		issues = append(issues, "Query SQL is required")
	}

	if query.Complexity < 1 || query.Complexity > 10 {
		issues = append(issues, "Query complexity must be between 1 and 10")
	}

	return issues
}

// GetQueriesByComplexity returns queries filtered by complexity range
func (ql *QueryLibrary) GetQueriesByComplexity(dbType DatabaseType, minComplexity, maxComplexity int) []Query {
	var results []Query

	if dbQueries, exists := ql.queries[dbType]; exists {
		for _, queryTypeQueries := range dbQueries {
			for _, query := range queryTypeQueries {
				if query.Complexity >= minComplexity && query.Complexity <= maxComplexity {
					results = append(results, query)
				}
			}
		}
	}

	return results
}

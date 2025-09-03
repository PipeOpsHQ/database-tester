package queries

import (
	"fmt"
	"math/rand"
	"time"
)

// QueryType represents different types of stress test queries
type QueryType string

const (
	QueryTypeSimple         QueryType = "simple"
	QueryTypeComplex        QueryType = "complex"
	QueryTypeRead           QueryType = "read"
	QueryTypeWrite          QueryType = "write"
	QueryTypeMixed          QueryType = "mixed"
	QueryTypeAnalytics      QueryType = "analytics"
	QueryTypeOLTP           QueryType = "oltp"
	QueryTypeJoin           QueryType = "join"
	QueryTypeAggregate      QueryType = "aggregate"
	QueryTypeRecursive      QueryType = "recursive"
	QueryTypeHeavyRead      QueryType = "heavy_read"
	QueryTypeHeavyWrite     QueryType = "heavy_write"
	QueryTypeBulkInsert     QueryType = "bulk_insert"
	QueryTypeBulkUpdate     QueryType = "bulk_update"
	QueryTypeTransaction    QueryType = "transaction"
	QueryTypeIndexStress    QueryType = "index_stress"
	QueryTypeConcurrent     QueryType = "concurrent"
	QueryTypeFullTextSearch QueryType = "fulltext_search"
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

	// MSSQL-specific heavy operations
	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "CTE Recursive Query",
		Description: "Common Table Expression with recursive operations",
		SQL: `WITH OrderHierarchy AS (
			-- Anchor member
			SELECT
				o.id,
				o.user_id,
				o.total_amount,
				o.order_date,
				1 as level,
				CAST(o.id AS VARCHAR(MAX)) as path
			FROM orders o
			WHERE o.parent_order_id IS NULL

			UNION ALL

			-- Recursive member
			SELECT
				o.id,
				o.user_id,
				o.total_amount,
				o.order_date,
				oh.level + 1,
				oh.path + '->' + CAST(o.id AS VARCHAR(MAX))
			FROM orders o
			INNER JOIN OrderHierarchy oh ON o.parent_order_id = oh.id
			WHERE oh.level < 5
		),
		OrderStats AS (
			SELECT
				level,
				COUNT(*) as order_count,
				SUM(total_amount) as level_revenue,
				AVG(total_amount) as avg_order_value,
				MIN(order_date) as earliest_order,
				MAX(order_date) as latest_order
			FROM OrderHierarchy
			GROUP BY level
		)
		SELECT
			os.*,
			SUM(os.level_revenue) OVER (ORDER BY os.level) as cumulative_revenue,
			PERCENT_RANK() OVER (ORDER BY os.order_count) as count_percentile
		FROM OrderStats os
		ORDER BY level`,
		Complexity: 10,
		Expected:   "Hierarchical order analysis with CTEs",
	})

	ql.addQuery(db, QueryTypeHeavyWrite, Query{
		Name:        "Merge Statement Operation",
		Description: "Complex MERGE statement for upsert operations",
		SQL: fmt.Sprintf(`WITH SourceData AS (
			SELECT %d as user_id, 'temp_user_%d' as username, 'temp_%d@test.com' as email, %d as age, %.2f as salary
			UNION ALL
			SELECT %d, 'temp_user_%d', 'temp_%d@test.com', %d, %.2f
			UNION ALL
			SELECT %d, 'temp_user_%d', 'temp_%d@test.com', %d, %.2f
		)
		MERGE users AS target
		USING SourceData AS source ON target.id = source.user_id
		WHEN MATCHED THEN
			UPDATE SET
				username = source.username,
				email = source.email,
				age = source.age,
				salary = source.salary,
				updated_at = GETDATE()
		WHEN NOT MATCHED THEN
			INSERT (username, email, age, salary, created_at, is_active)
			VALUES (source.username, source.email, source.age, source.salary, GETDATE(), 1)
		OUTPUT $action, inserted.id, inserted.username;`,
			rand.Intn(1000), rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(1000), rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(1000), rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), 30000+rand.Float64()*70000),
		Complexity: 8,
		Expected:   "MERGE operation with OUTPUT clause",
	})

	ql.addQuery(db, QueryTypeIndexStress, Query{
		Name:        "Dynamic SQL with Hints",
		Description: "Query with table hints and index optimization",
		SQL: `SELECT TOP 1000
			u.id,
			u.username,
			u.email,
			COUNT(o.id) as order_count,
			SUM(o.total_amount) as total_spent,
			MAX(o.order_date) as last_order_date
		FROM users u WITH (INDEX(IX_Users_Username), NOLOCK)
		LEFT JOIN orders o WITH (INDEX(IX_Orders_UserId_Date), NOLOCK)
			ON u.id = o.user_id
			AND o.order_date >= DATEADD(MONTH, -6, GETDATE())
		WHERE u.is_active = 1
			AND u.created_at >= DATEADD(YEAR, -1, GETDATE())
		GROUP BY u.id, u.username, u.email
		HAVING COUNT(o.id) > 0
		ORDER BY total_spent DESC
		OPTION (OPTIMIZE FOR (@param1 = 1000))`,
		Complexity: 7,
		Expected:   "Optimized query with table hints",
	})

	// Heavy Read Queries for PostgreSQL
	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Complex Window Functions",
		Description: "Advanced analytics with multiple window functions",
		SQL: `SELECT
			u.id,
			u.username,
			u.created_at,
			COUNT(o.id) OVER (PARTITION BY u.id) as total_orders,
			SUM(o.total_amount) OVER (PARTITION BY u.id) as lifetime_value,
			AVG(o.total_amount) OVER (PARTITION BY u.id) as avg_order_value,
			ROW_NUMBER() OVER (ORDER BY SUM(o.total_amount) DESC) as value_rank,
			DENSE_RANK() OVER (ORDER BY COUNT(o.id) DESC) as order_rank,
			LAG(o.order_date) OVER (PARTITION BY u.id ORDER BY o.order_date) as prev_order,
			LEAD(o.order_date) OVER (PARTITION BY u.id ORDER BY o.order_date) as next_order,
			FIRST_VALUE(o.order_date) OVER (PARTITION BY u.id ORDER BY o.order_date) as first_order,
			LAST_VALUE(o.order_date) OVER (PARTITION BY u.id ORDER BY o.order_date
				RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as last_order,
			NTILE(10) OVER (ORDER BY SUM(o.total_amount)) as value_decile
		FROM users u
		LEFT JOIN orders o ON u.id = o.user_id
		WHERE u.created_at >= NOW() - INTERVAL '1 year'
		GROUP BY u.id, u.username, u.created_at, o.order_date, o.total_amount
		ORDER BY lifetime_value DESC NULLS LAST`,
		Complexity: 10,
		Expected:   "Advanced window function analytics",
	})

	// PostgreSQL-specific heavy operations
	ql.addQuery(db, QueryTypeHeavyWrite, Query{
		Name:        "Bulk Insert with RETURNING",
		Description: "Mass insert operation with returned data",
		SQL: fmt.Sprintf(`WITH bulk_data AS (
			INSERT INTO users (username, email, first_name, last_name, age, salary, created_at)
			VALUES
				('pg_user_%d_1', 'pg_%d_1@test.com', 'PG', 'User1', %d, %.2f, NOW()),
				('pg_user_%d_2', 'pg_%d_2@test.com', 'PG', 'User2', %d, %.2f, NOW()),
				('pg_user_%d_3', 'pg_%d_3@test.com', 'PG', 'User3', %d, %.2f, NOW()),
				('pg_user_%d_4', 'pg_%d_4@test.com', 'PG', 'User4', %d, %.2f, NOW()),
				('pg_user_%d_5', 'pg_%d_5@test.com', 'PG', 'User5', %d, %.2f, NOW())
			RETURNING id, username, email, created_at
		)
		SELECT
			bd.*,
			'bulk_insert_' || extract(epoch from bd.created_at) as batch_id
		FROM bulk_data bd`,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000),
		Complexity: 7,
		Expected:   "Bulk insert with CTE and RETURNING clause",
	})

	// PostgreSQL JSON operations
	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "JSON Aggregation Query",
		Description: "Complex JSON operations and aggregations",
		SQL: `SELECT
			DATE_TRUNC('day', o.order_date) as order_day,
			COUNT(*) as total_orders,
			JSON_AGG(
				JSON_BUILD_OBJECT(
					'order_id', o.id,
					'user_id', o.user_id,
					'total', o.total_amount,
					'items', (
						SELECT JSON_AGG(
							JSON_BUILD_OBJECT(
								'product_id', oi.product_id,
								'quantity', oi.quantity,
								'price', oi.unit_price
							)
						)
						FROM order_items oi
						WHERE oi.order_id = o.id
					)
				) ORDER BY o.total_amount DESC
			) as orders_json,
			JSON_OBJECT_AGG(o.status, COUNT(*)) as status_counts
		FROM orders o
		WHERE o.order_date >= NOW() - INTERVAL '30 days'
		GROUP BY DATE_TRUNC('day', o.order_date)
		ORDER BY order_day DESC`,
		Complexity: 9,
		Expected:   "JSON aggregated order data with nested structures",
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

	// Heavy Read Queries
	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Large Table Scan",
		Description: "Full table scan with filtering - stress test for large datasets",
		SQL: `SELECT u.*, o.total_amount, o.order_date, oi.quantity, p.product_name
			   FROM users u
			   LEFT JOIN orders o ON u.id = o.user_id
			   LEFT JOIN order_items oi ON o.id = oi.order_id
			   LEFT JOIN products p ON oi.product_id = p.id
			   WHERE u.created_at >= DATE_SUB(NOW(), INTERVAL 1 YEAR)
			   AND (u.age BETWEEN 25 AND 65 OR u.salary > 50000)
			   ORDER BY u.last_login DESC, o.order_date DESC`,
		Complexity: 9,
		Expected:   "Large dataset with multiple joins and filtering",
	})

	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Complex Analytics Query",
		Description: "Resource-intensive analytical query with multiple aggregations",
		SQL: `SELECT
			DATE(o.order_date) as date,
			COUNT(DISTINCT o.id) as orders,
			COUNT(DISTINCT o.user_id) as customers,
			SUM(o.total_amount) as revenue,
			AVG(o.total_amount) as avg_order,
			MIN(o.total_amount) as min_order,
			MAX(o.total_amount) as max_order,
			STDDEV(o.total_amount) as stddev_order,
			COUNT(DISTINCT oi.product_id) as products_sold,
			SUM(oi.quantity) as items_sold,
			COUNT(CASE WHEN o.status = 'completed' THEN 1 END) as completed_orders,
			COUNT(CASE WHEN o.status = 'cancelled' THEN 1 END) as cancelled_orders,
			(COUNT(CASE WHEN o.status = 'completed' THEN 1 END) * 100.0 / COUNT(*)) as completion_rate
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		WHERE o.order_date >= DATE_SUB(NOW(), INTERVAL 90 DAY)
		GROUP BY DATE(o.order_date)
		HAVING COUNT(*) > 10
		ORDER BY date DESC`,
		Complexity: 10,
		Expected:   "Daily analytics with multiple metrics and calculations",
	})

	// Heavy Write Queries
	ql.addQuery(db, QueryTypeBulkInsert, Query{
		Name:        "Bulk User Insert",
		Description: "Insert multiple users in single query - write stress test",
		SQL: fmt.Sprintf(`INSERT INTO users (username, email, first_name, last_name, age, salary, created_at, is_active) VALUES
			   ('bulk_user_%d_1', 'bulk_%d_1@test.com', 'Bulk', 'User1', %d, %.2f, NOW(), 1),
			   ('bulk_user_%d_2', 'bulk_%d_2@test.com', 'Bulk', 'User2', %d, %.2f, NOW(), 1),
			   ('bulk_user_%d_3', 'bulk_%d_3@test.com', 'Bulk', 'User3', %d, %.2f, NOW(), 1),
			   ('bulk_user_%d_4', 'bulk_%d_4@test.com', 'Bulk', 'User4', %d, %.2f, NOW(), 1),
			   ('bulk_user_%d_5', 'bulk_%d_5@test.com', 'Bulk', 'User5', %d, %.2f, NOW(), 1)`,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), 30000+rand.Float64()*70000),
		Complexity: 6,
		Expected:   "Multiple users inserted in single transaction",
	})

	ql.addQuery(db, QueryTypeBulkUpdate, Query{
		Name:        "Mass Price Update",
		Description: "Update multiple records with complex calculations",
		SQL: `UPDATE products p
			   JOIN (
				   SELECT product_id,
						  AVG(oi.unit_price) as avg_price,
						  COUNT(*) as order_count
				   FROM order_items oi
				   JOIN orders o ON oi.order_id = o.id
				   WHERE o.order_date >= DATE_SUB(NOW(), INTERVAL 30 DAY)
				   GROUP BY product_id
				   HAVING COUNT(*) > 5
			   ) stats ON p.id = stats.product_id
			   SET p.price = CASE
				   WHEN stats.order_count > 50 THEN stats.avg_price * 1.1
				   WHEN stats.order_count > 20 THEN stats.avg_price * 1.05
				   ELSE stats.avg_price
			   END,
			   p.updated_at = NOW(),
			   p.last_price_update = NOW()
			   WHERE p.is_active = 1`,
		Complexity: 8,
		Expected:   "Dynamic pricing updates based on sales data",
	})

	// Transaction-based queries
	ql.addQuery(db, QueryTypeTransaction, Query{
		Name:        "Order Processing Transaction",
		Description: "Complex multi-table transaction simulating order processing",
		SQL: `START TRANSACTION;

			   INSERT INTO orders (user_id, total_amount, status, order_date)
			   VALUES (FLOOR(1 + RAND() * 1000), ROUND(50 + RAND() * 500, 2), 'pending', NOW());

			   SET @order_id = LAST_INSERT_ID();

			   INSERT INTO order_items (order_id, product_id, quantity, unit_price) VALUES
			   (@order_id, FLOOR(1 + RAND() * 100), FLOOR(1 + RAND() * 5), ROUND(10 + RAND() * 90, 2)),
			   (@order_id, FLOOR(1 + RAND() * 100), FLOOR(1 + RAND() * 3), ROUND(15 + RAND() * 85, 2));

			   UPDATE products p
			   JOIN order_items oi ON p.id = oi.product_id
			   SET p.stock_quantity = p.stock_quantity - oi.quantity
			   WHERE oi.order_id = @order_id AND p.stock_quantity >= oi.quantity;

			   UPDATE orders SET status = 'confirmed' WHERE id = @order_id;

			   COMMIT;`,
		Complexity: 9,
		Expected:   "Complete order processing with inventory updates",
	})

	// Index stress queries
	ql.addQuery(db, QueryTypeIndexStress, Query{
		Name:        "Index Performance Test",
		Description: "Query designed to stress test database indexes",
		SQL: `SELECT u.id, u.username, u.email,
			   COUNT(o.id) as order_count,
			   SUM(o.total_amount) as total_spent,
			   MAX(o.order_date) as last_order,
			   MIN(o.order_date) as first_order,
			   DATEDIFF(MAX(o.order_date), MIN(o.order_date)) as customer_lifetime_days
			   FROM users u
			   FORCE INDEX (idx_username, idx_email)
			   LEFT JOIN orders o FORCE INDEX (idx_user_date) ON u.id = o.user_id
			   WHERE u.created_at BETWEEN DATE_SUB(NOW(), INTERVAL 6 MONTH) AND NOW()
			   AND u.is_active = 1
			   AND (u.username LIKE 'test%' OR u.email LIKE '%@test.com')
			   GROUP BY u.id, u.username, u.email
			   HAVING order_count > 0
			   ORDER BY total_spent DESC, last_order DESC
			   LIMIT 100`,
		Complexity: 7,
		Expected:   "Index utilization test with forced index usage",
	})

	// Mixed workload queries
	ql.addQuery(db, QueryTypeMixed, Query{
		Name:        "Read-Write Mixed Operation",
		Description: "Combined read and write operations in single query",
		SQL: `UPDATE user_stats us
			   JOIN (
				   SELECT u.id,
						  COUNT(o.id) as new_order_count,
						  COALESCE(SUM(o.total_amount), 0) as new_total_spent,
						  COALESCE(AVG(o.total_amount), 0) as new_avg_order
				   FROM users u
				   LEFT JOIN orders o ON u.id = o.user_id
				   AND o.order_date >= DATE_SUB(NOW(), INTERVAL 1 DAY)
				   WHERE u.is_active = 1
				   GROUP BY u.id
			   ) calc ON us.user_id = calc.id
			   SET us.total_orders = us.total_orders + calc.new_order_count,
				   us.total_spent = us.total_spent + calc.new_total_spent,
				   us.avg_order_value = CASE
					   WHEN us.total_orders + calc.new_order_count > 0
					   THEN (us.total_spent + calc.new_total_spent) / (us.total_orders + calc.new_order_count)
					   ELSE 0
				   END,
				   us.last_updated = NOW()
			   WHERE calc.new_order_count > 0`,
		Complexity: 8,
		Expected:   "User statistics update with real-time calculations",
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

	// Simple operations
	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Basic GET",
		Description: "Simple key-value retrieval",
		SQL:         "GET test-key-" + fmt.Sprintf("%d", rand.Intn(1000)),
		Complexity:  1,
		Expected:    "Key value or nil",
	})

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Key Existence Check",
		Description: "Check if key exists",
		SQL:         "EXISTS stress-test-key-" + fmt.Sprintf("%d", rand.Intn(100)),
		Complexity:  1,
		Expected:    "1 if exists, 0 if not",
	})

	// Write operations
	ql.addQuery(db, QueryTypeWrite, Query{
		Name:        "SET with Expiry",
		Description: "Set key with expiration",
		SQL:         fmt.Sprintf("SETEX stress-key-%d 3600 'value-%d-%d'", rand.Intn(1000), rand.Intn(1000), time.Now().Unix()),
		Complexity:  2,
		Expected:    "OK response",
	})

	ql.addQuery(db, QueryTypeWrite, Query{
		Name:        "Hash Set Multiple",
		Description: "Set multiple hash fields",
		SQL: fmt.Sprintf("HMSET user:%d name 'User%d' email 'user%d@test.com' age %d score %.2f",
			rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), rand.Float64()*1000),
		Complexity: 3,
		Expected:   "OK response",
	})

	// Heavy read operations
	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Scan All Keys",
		Description: "Scan through all keys with pattern",
		SQL:         "SCAN 0 MATCH stress-* COUNT 1000",
		Complexity:  7,
		Expected:    "Cursor and key list",
	})

	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Multi-Key Retrieval",
		Description: "Get multiple keys in single command",
		SQL: fmt.Sprintf("MGET stress-key-%d stress-key-%d stress-key-%d stress-key-%d stress-key-%d",
			rand.Intn(1000), rand.Intn(1000), rand.Intn(1000), rand.Intn(1000), rand.Intn(1000)),
		Complexity: 4,
		Expected:   "Array of values",
	})

	// Heavy write operations
	ql.addQuery(db, QueryTypeBulkInsert, Query{
		Name:        "List Bulk Push",
		Description: "Push multiple items to list",
		SQL: fmt.Sprintf("LPUSH bulk-list:%d 'item1-%d' 'item2-%d' 'item3-%d' 'item4-%d' 'item5-%d'",
			rand.Intn(100), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000)),
		Complexity: 5,
		Expected:   "New length of list",
	})

	ql.addQuery(db, QueryTypeBulkInsert, Query{
		Name:        "Set Bulk Add",
		Description: "Add multiple members to set",
		SQL: fmt.Sprintf("SADD bulk-set:%d 'member1-%d' 'member2-%d' 'member3-%d' 'member4-%d' 'member5-%d'",
			rand.Intn(100), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000), rand.Intn(10000)),
		Complexity: 5,
		Expected:   "Number of added members",
	})

	// Index stress (sorted sets)
	ql.addQuery(db, QueryTypeIndexStress, Query{
		Name:        "Sorted Set Range Query",
		Description: "Range queries on sorted sets",
		SQL: fmt.Sprintf("ZRANGEBYSCORE leaderboard:%d %d %d WITHSCORES LIMIT 0 100",
			rand.Intn(10), rand.Intn(1000), 500+rand.Intn(1000)),
		Complexity: 6,
		Expected:   "Scored members in range",
	})

	ql.addQuery(db, QueryTypeIndexStress, Query{
		Name:        "Sorted Set Rank Operations",
		Description: "Get rank and score operations",
		SQL:         fmt.Sprintf("ZRANK leaderboard:%d user:%d", rand.Intn(10), rand.Intn(1000)),
		Complexity:  4,
		Expected:    "Member rank or nil",
	})

	// Transaction operations
	ql.addQuery(db, QueryTypeTransaction, Query{
		Name:        "Multi-Command Transaction",
		Description: "Atomic multi-command transaction",
		SQL: fmt.Sprintf("MULTI|SET txn-key:%d 'value'|INCR txn-counter:%d|EXPIRE txn-key:%d 3600|EXEC",
			rand.Intn(1000), rand.Intn(100), rand.Intn(1000)),
		Complexity: 7,
		Expected:   "Transaction results array",
	})

	// Concurrent operations
	ql.addQuery(db, QueryTypeConcurrent, Query{
		Name:        "Atomic Counter",
		Description: "Atomic increment operations for concurrency testing",
		SQL:         fmt.Sprintf("INCR concurrent-counter:%d", rand.Intn(10)),
		Complexity:  3,
		Expected:    "Incremented value",
	})

	ql.addQuery(db, QueryTypeConcurrent, Query{
		Name:        "List Pop Operations",
		Description: "Blocking pop operations for queue simulation",
		SQL:         fmt.Sprintf("LPOP work-queue:%d", rand.Intn(5)),
		Complexity:  4,
		Expected:    "Popped element or nil",
	})

	// Complex operations
	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Pipeline Operations",
		Description: "Multiple operations in pipeline",
		SQL:         "PIPELINE:SET|GET|INCR|EXPIRE", // Special format for Redis pipeline
		Complexity:  5,
		Expected:    "Multiple operation results",
	})

	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Lua Script Execution",
		Description: "Execute Lua script for complex operations",
		SQL: fmt.Sprintf("EVAL \"local key = KEYS[1] local val = redis.call('GET', key) if val then return redis.call('INCR', key) else redis.call('SET', key, 1) return 1 end\" 1 script-counter:%d",
			rand.Intn(100)),
		Complexity: 8,
		Expected:   "Script execution result",
	})

	// Analytics operations
	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "HyperLogLog Cardinality",
		Description: "Approximate cardinality counting",
		SQL: fmt.Sprintf("PFADD unique-visitors:%d user:%d",
			rand.Intn(10), rand.Intn(10000)),
		Complexity: 4,
		Expected:   "1 if new element, 0 if exists",
	})

	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "Geospatial Operations",
		Description: "Add and query geospatial data",
		SQL: fmt.Sprintf("GEOADD locations:%d %.6f %.6f 'point:%d'",
			rand.Intn(10), -180+rand.Float64()*360, -90+rand.Float64()*180, rand.Intn(1000)),
		Complexity: 5,
		Expected:   "Number of added elements",
	})

	// Mixed workload
	ql.addQuery(db, QueryTypeMixed, Query{
		Name:        "Cache Simulation",
		Description: "Simulate cache miss/hit pattern with fallback",
		SQL: fmt.Sprintf("GET cache:user:%d|SET cache:user:%d 'computed-value-%d' EX 300",
			rand.Intn(1000), rand.Intn(1000), rand.Intn(10000)),
		Complexity: 6,
		Expected:   "Cache value or set operation",
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

	// Simple operations
	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Find Documents",
		Description: "Simple document find operation",
		SQL:         `db.testcollection.find({}).limit(10)`,
		Complexity:  1,
		Expected:    "List of documents",
	})

	ql.addQuery(db, QueryTypeSimple, Query{
		Name:        "Count Documents",
		Description: "Count documents in collection",
		SQL:         `db.testcollection.countDocuments({})`,
		Complexity:  1,
		Expected:    "Document count",
	})

	// Write operations
	ql.addQuery(db, QueryTypeWrite, Query{
		Name:        "Insert Document",
		Description: "Insert single document with random data",
		SQL: fmt.Sprintf(`db.testcollection.insertOne({
			"name": "TestUser%d",
			"email": "test%d@example.com",
			"age": %d,
			"score": %.2f,
			"tags": ["tag%d", "tag%d"],
			"createdAt": new Date(),
			"metadata": {
				"source": "stress_test",
				"version": "1.0",
				"sessionId": "%d"
			}
		})`,
			rand.Intn(100000), rand.Intn(100000), 20+rand.Intn(50), rand.Float64()*1000,
			rand.Intn(100), rand.Intn(100), rand.Intn(10000)),
		Complexity: 3,
		Expected:   "Insert result with ObjectId",
	})

	// Heavy read operations
	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Complex Find with Sorting",
		Description: "Large result set with complex filtering and sorting",
		SQL: `db.testcollection.find({
			"$and": [
				{"age": {"$gte": 18, "$lte": 65}},
				{"score": {"$gt": 100}},
				{"$or": [
					{"tags": {"$in": ["premium", "gold"]}},
					{"metadata.source": "web"}
				]}
			]
		}).sort({"score": -1, "createdAt": -1}).limit(1000)`,
		Complexity: 8,
		Expected:   "Filtered and sorted documents",
	})

	ql.addQuery(db, QueryTypeHeavyRead, Query{
		Name:        "Text Search Query",
		Description: "Full-text search across multiple fields",
		SQL: `db.testcollection.find({
			"$text": {"$search": "user test premium"}
		}, {
			"score": {"$meta": "textScore"}
		}).sort({"score": {"$meta": "textScore"}}).limit(100)`,
		Complexity: 6,
		Expected:   "Text search results with scores",
	})

	// Bulk operations
	ql.addQuery(db, QueryTypeBulkInsert, Query{
		Name:        "Bulk Insert Multiple",
		Description: "Insert multiple documents in single operation",
		SQL: fmt.Sprintf(`db.testcollection.insertMany([
			{
				"name": "BulkUser%d_1",
				"email": "bulk%d_1@test.com",
				"age": %d,
				"score": %.2f,
				"batch": "%d",
				"createdAt": new Date()
			},
			{
				"name": "BulkUser%d_2",
				"email": "bulk%d_2@test.com",
				"age": %d,
				"score": %.2f,
				"batch": "%d",
				"createdAt": new Date()
			},
			{
				"name": "BulkUser%d_3",
				"email": "bulk%d_3@test.com",
				"age": %d,
				"score": %.2f,
				"batch": "%d",
				"createdAt": new Date()
			}
		])`,
			rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), rand.Float64()*1000, rand.Intn(1000),
			rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), rand.Float64()*1000, rand.Intn(1000),
			rand.Intn(10000), rand.Intn(10000), 20+rand.Intn(50), rand.Float64()*1000, rand.Intn(1000)),
		Complexity: 5,
		Expected:   "Bulk insert result with multiple ObjectIds",
	})

	ql.addQuery(db, QueryTypeBulkUpdate, Query{
		Name:        "Bulk Update Many",
		Description: "Update multiple documents with complex operations",
		SQL: `db.testcollection.updateMany(
			{"score": {"$lt": 500}},
			{
				"$inc": {"score": 10, "updateCount": 1},
				"$set": {"lastUpdated": new Date(), "status": "updated"},
				"$addToSet": {"tags": "bulk_updated"}
			}
		)`,
		Complexity: 6,
		Expected:   "Update result with matched and modified counts",
	})

	// Index stress operations
	ql.addQuery(db, QueryTypeIndexStress, Query{
		Name:        "Compound Index Query",
		Description: "Query utilizing compound indexes with multiple conditions",
		SQL: `db.testcollection.find({
			"age": {"$gte": 25, "$lte": 45},
			"score": {"$gte": 300},
			"tags": {"$in": ["premium", "gold", "platinum"]},
			"createdAt": {"$gte": new Date(Date.now() - 30*24*60*60*1000)}
		}).hint({"age": 1, "score": -1, "createdAt": -1}).limit(500)`,
		Complexity: 7,
		Expected:   "Documents using compound index",
	})

	// Complex aggregation operations
	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Advanced Aggregation Pipeline",
		Description: "Complex aggregation with multiple stages and lookups",
		SQL: `db.testcollection.aggregate([
			{"$match": {"age": {"$gte": 18}}},
			{"$addFields": {
				"ageGroup": {
					"$switch": {
						"branches": [
							{"case": {"$lt": ["$age", 25]}, "then": "young"},
							{"case": {"$lt": ["$age", 45]}, "then": "adult"},
							{"case": {"$lt": ["$age", 65]}, "then": "mature"}
						],
						"default": "senior"
					}
				},
				"scoreCategory": {
					"$cond": {
						"if": {"$gte": ["$score", 800]}, "then": "high",
						"else": {"$cond": {"if": {"$gte": ["$score", 500]}, "then": "medium", "else": "low"}}
					}
				}
			}},
			{"$group": {
				"_id": {"ageGroup": "$ageGroup", "scoreCategory": "$scoreCategory"},
				"count": {"$sum": 1},
				"avgScore": {"$avg": "$score"},
				"maxScore": {"$max": "$score"},
				"minScore": {"$min": "$score"},
				"users": {"$push": {"name": "$name", "email": "$email"}}
			}},
			{"$sort": {"_id.ageGroup": 1, "_id.scoreCategory": 1}},
			{"$project": {
				"ageGroup": "$_id.ageGroup",
				"scoreCategory": "$_id.scoreCategory",
				"statistics": {
					"count": "$count",
					"avgScore": {"$round": ["$avgScore", 2]},
					"maxScore": "$maxScore",
					"minScore": "$minScore",
					"scoreRange": {"$subtract": ["$maxScore", "$minScore"]}
				},
				"sampleUsers": {"$slice": ["$users", 3]},
				"_id": 0
			}}
		])`,
		Complexity: 10,
		Expected:   "Complex aggregated user analytics",
	})

	ql.addQuery(db, QueryTypeComplex, Query{
		Name:        "Geospatial Aggregation",
		Description: "Geospatial queries with aggregation pipeline",
		SQL: fmt.Sprintf(`db.locations.aggregate([
			{"$geoNear": {
				"near": {"type": "Point", "coordinates": [%.6f, %.6f]},
				"distanceField": "distance",
				"maxDistance": 10000,
				"spherical": true
			}},
			{"$group": {
				"_id": {
					"$toInt": {"$divide": ["$distance", 1000]}
				},
				"count": {"$sum": 1},
				"avgDistance": {"$avg": "$distance"},
				"locations": {"$push": "$name"}
			}},
			{"$sort": {"_id": 1}},
			{"$project": {
				"kmRange": {"$concat": [{"$toString": "$_id"}, "-", {"$toString": {"$add": ["$_id", 1]}}, "km"]},
				"count": 1,
				"avgDistance": {"$round": ["$avgDistance", 2]},
				"sampleLocations": {"$slice": ["$locations", 5]},
				"_id": 0
			}}
		])`,
			-180+rand.Float64()*360, -90+rand.Float64()*180),
		Complexity: 9,
		Expected:   "Geospatial distance analysis",
	})

	// Transaction simulation
	ql.addQuery(db, QueryTypeTransaction, Query{
		Name:        "Multi-Document Transaction",
		Description: "Simulate multi-document ACID transaction",
		SQL: fmt.Sprintf(`
		session = db.getMongo().startSession();
		session.startTransaction();
		try {
			db.testcollection.insertOne({
				"name": "TxnUser%d",
				"email": "txn%d@test.com",
				"balance": %.2f,
				"transactionId": "%d"
			}, {session: session});

			db.transactions.insertOne({
				"userId": "TxnUser%d",
				"amount": %.2f,
				"type": "credit",
				"timestamp": new Date(),
				"transactionId": "%d"
			}, {session: session});

			session.commitTransaction();
		} catch (error) {
			session.abortTransaction();
			throw error;
		} finally {
			session.endSession();
		}`,
			rand.Intn(10000), rand.Intn(10000), 100+rand.Float64()*900, rand.Intn(100000),
			rand.Intn(10000), 50+rand.Float64()*200, rand.Intn(100000)),
		Complexity: 8,
		Expected:   "Transaction completion or rollback",
	})

	// Analytics operations
	ql.addQuery(db, QueryTypeAnalytics, Query{
		Name:        "Time Series Analytics",
		Description: "Time-based analytics with date bucketing",
		SQL: `db.testcollection.aggregate([
			{"$match": {
				"createdAt": {"$gte": new Date(Date.now() - 30*24*60*60*1000)}
			}},
			{"$group": {
				"_id": {
					"year": {"$year": "$createdAt"},
					"month": {"$month": "$createdAt"},
					"day": {"$dayOfMonth": "$createdAt"},
					"hour": {"$hour": "$createdAt"}
				},
				"count": {"$sum": 1},
				"avgScore": {"$avg": "$score"},
				"uniqueUsers": {"$addToSet": "$name"}
			}},
			{"$addFields": {
				"uniqueUserCount": {"$size": "$uniqueUsers"},
				"date": {
					"$dateFromParts": {
						"year": "$_id.year",
						"month": "$_id.month",
						"day": "$_id.day",
						"hour": "$_id.hour"
					}
				}
			}},
			{"$sort": {"date": -1}},
			{"$limit": 100}
		])`,
		Complexity: 8,
		Expected:   "Hourly activity analytics",
	})

	// Mixed workload
	ql.addQuery(db, QueryTypeMixed, Query{
		Name:        "Read-Write Mixed Operation",
		Description: "Combined read, update, and insert operations",
		SQL: fmt.Sprintf(`
		var user = db.testcollection.findOne({"name": "TestUser%d"});
		if (user) {
			db.testcollection.updateOne(
				{"_id": user._id},
				{
					"$inc": {"score": 1, "accessCount": 1},
					"$set": {"lastAccessed": new Date()}
				}
			);
		} else {
			db.testcollection.insertOne({
				"name": "TestUser%d",
				"email": "newuser%d@test.com",
				"age": %d,
				"score": 1,
				"accessCount": 1,
				"createdAt": new Date(),
				"lastAccessed": new Date()
			});
		}`,
			rand.Intn(1000), rand.Intn(1000), rand.Intn(1000), 20+rand.Intn(50)),
		Complexity: 7,
		Expected:   "User upsert with access tracking",
	})

	// Concurrent operations
	ql.addQuery(db, QueryTypeConcurrent, Query{
		Name:        "Atomic Increment",
		Description: "Atomic increment operations for concurrency testing",
		SQL: fmt.Sprintf(`db.counters.findOneAndUpdate(
			{"_id": "concurrent_counter_%d"},
			{"$inc": {"value": 1}},
			{"upsert": true, "returnDocument": "after"}
		)`, rand.Intn(10)),
		Complexity: 4,
		Expected:   "Updated counter document",
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

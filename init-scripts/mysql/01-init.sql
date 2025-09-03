-- MySQL Initialization Script for Database Stress Testing
-- This script creates tables and sample data for stress testing

-- Ensure we're using the correct database
USE testdb;

-- Create a simple users table
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    age INT,
    salary DECIMAL(10,2),
    INDEX idx_username (username),
    INDEX idx_email (email),
    INDEX idx_created_at (created_at),
    INDEX idx_active_users (is_active, created_at)
);

-- Create an orders table for complex join testing
CREATE TABLE IF NOT EXISTS orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    order_number VARCHAR(20) NOT NULL UNIQUE,
    total_amount DECIMAL(12,2) NOT NULL,
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shipped_date TIMESTAMP NULL,
    notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_order_number (order_number),
    INDEX idx_status (status),
    INDEX idx_order_date (order_date),
    INDEX idx_user_status (user_id, status)
);

-- Create order items table for more complex relationships
CREATE TABLE IF NOT EXISTS order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT,
    product_name VARCHAR(100) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(10,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    INDEX idx_order_id (order_id),
    INDEX idx_product_name (product_name)
);

-- Create a products table for inventory testing
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    sku VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock_quantity INT NOT NULL DEFAULT 0,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    is_available BOOLEAN DEFAULT TRUE,
    weight DECIMAL(8,3),
    dimensions JSON,
    INDEX idx_sku (sku),
    INDEX idx_name (name),
    INDEX idx_category (category),
    INDEX idx_price (price),
    INDEX idx_stock (stock_quantity),
    INDEX idx_available (is_available),
    FULLTEXT idx_description (description)
);

-- Create a log table for high-volume insert testing
CREATE TABLE IF NOT EXISTS activity_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(50) NOT NULL,
    details JSON,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR(100),
    INDEX idx_user_id (user_id),
    INDEX idx_action (action),
    INDEX idx_created_at (created_at),
    INDEX idx_session_id (session_id)
);

-- Create a view for complex queries
CREATE OR REPLACE VIEW user_order_summary AS
SELECT
    u.id,
    u.username,
    u.email,
    u.first_name,
    u.last_name,
    COUNT(o.id) as total_orders,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    COALESCE(AVG(o.total_amount), 0) as avg_order_value,
    MAX(o.order_date) as last_order_date,
    u.created_at as user_since
FROM users u
LEFT JOIN orders o ON u.id = o.user_id AND o.status != 'cancelled'
GROUP BY u.id, u.username, u.email, u.first_name, u.last_name, u.created_at;

-- Insert sample users
INSERT IGNORE INTO users (username, email, first_name, last_name, age, salary) VALUES
('john_doe', 'john.doe@example.com', 'John', 'Doe', 28, 50000.00),
('jane_smith', 'jane.smith@example.com', 'Jane', 'Smith', 32, 65000.00),
('bob_johnson', 'bob.johnson@example.com', 'Bob', 'Johnson', 45, 75000.00),
('alice_brown', 'alice.brown@example.com', 'Alice', 'Brown', 29, 55000.00),
('charlie_davis', 'charlie.davis@example.com', 'Charlie', 'Davis', 38, 80000.00),
('diana_wilson', 'diana.wilson@example.com', 'Diana', 'Wilson', 26, 48000.00),
('edward_miller', 'edward.miller@example.com', 'Edward', 'Miller', 35, 72000.00),
('fiona_taylor', 'fiona.taylor@example.com', 'Fiona', 'Taylor', 31, 58000.00),
('george_anderson', 'george.anderson@example.com', 'George', 'Anderson', 42, 85000.00),
('helen_thomas', 'helen.thomas@example.com', 'Helen', 'Thomas', 27, 52000.00);

-- Insert sample products
INSERT IGNORE INTO products (sku, name, description, price, stock_quantity, category, weight, dimensions) VALUES
('LAPTOP-001', 'Gaming Laptop Pro', 'High-performance gaming laptop with RTX graphics', 1299.99, 25, 'Electronics', 2.5, '{"width": 35.6, "height": 2.3, "depth": 25.1}'),
('PHONE-001', 'Smartphone X', 'Latest smartphone with advanced camera', 899.99, 150, 'Electronics', 0.18, '{"width": 7.1, "height": 0.8, "depth": 14.7}'),
('BOOK-001', 'Programming Guide', 'Complete guide to modern programming', 49.99, 200, 'Books', 0.8, '{"width": 15.2, "height": 2.3, "depth": 22.9}'),
('CHAIR-001', 'Ergonomic Office Chair', 'Comfortable office chair with lumbar support', 299.99, 50, 'Furniture', 15.5, '{"width": 66, "height": 117, "depth": 66}'),
('DESK-001', 'Standing Desk', 'Adjustable height standing desk', 599.99, 20, 'Furniture', 35.2, '{"width": 120, "height": 75, "depth": 60}'),
('MOUSE-001', 'Wireless Gaming Mouse', 'High-precision wireless gaming mouse', 79.99, 100, 'Electronics', 0.12, '{"width": 6.8, "height": 4.2, "depth": 12.5}'),
('KEYBOARD-001', 'Mechanical Keyboard', 'RGB mechanical keyboard', 149.99, 75, 'Electronics', 1.2, '{"width": 44, "height": 3.5, "depth": 13.2}'),
('MONITOR-001', '4K Gaming Monitor', '27-inch 4K gaming monitor with HDR', 449.99, 30, 'Electronics', 6.8, '{"width": 61.4, "height": 45.7, "depth": 21.0}'),
('HEADSET-001', 'Wireless Headset', 'Noise-cancelling wireless headset', 199.99, 60, 'Electronics', 0.35, '{"width": 19.5, "height": 20.3, "depth": 8.9}'),
('TABLET-001', 'Professional Tablet', '12-inch professional tablet for creative work', 799.99, 40, 'Electronics', 0.68, '{"width": 28.1, "height": 0.7, "depth": 20.3}'};

-- Insert sample orders
INSERT IGNORE INTO orders (user_id, order_number, total_amount, status, order_date) VALUES
(1, 'ORD-2023-001', 1299.99, 'delivered', '2023-11-01 10:30:00'),
(2, 'ORD-2023-002', 949.98, 'delivered', '2023-11-02 14:15:00'),
(3, 'ORD-2023-003', 749.98, 'shipped', '2023-11-15 09:45:00'),
(4, 'ORD-2023-004', 299.99, 'processing', '2023-11-20 16:20:00'),
(1, 'ORD-2023-005', 229.98, 'delivered', '2023-11-25 11:10:00'),
(5, 'ORD-2023-006', 599.99, 'pending', '2023-12-01 08:30:00'),
(2, 'ORD-2023-007', 199.99, 'shipped', '2023-12-02 13:45:00'),
(6, 'ORD-2023-008', 1099.98, 'processing', '2023-12-05 15:20:00'),
(7, 'ORD-2023-009', 449.99, 'delivered', '2023-12-08 12:00:00'),
(8, 'ORD-2023-010', 849.98, 'shipped', '2023-12-10 10:15:00');

-- Insert sample order items
INSERT IGNORE INTO order_items (order_id, product_name, quantity, unit_price) VALUES
(1, 'Gaming Laptop Pro', 1, 1299.99),
(2, 'Smartphone X', 1, 899.99),
(2, 'Programming Guide', 1, 49.99),
(3, 'Standing Desk', 1, 599.99),
(3, 'Mechanical Keyboard', 1, 149.99),
(4, 'Ergonomic Office Chair', 1, 299.99),
(5, 'Wireless Gaming Mouse', 1, 79.99),
(5, 'Mechanical Keyboard', 1, 149.99),
(6, 'Standing Desk', 1, 599.99),
(7, 'Wireless Headset', 1, 199.99),
(8, 'Gaming Laptop Pro', 1, 1299.99),
(9, '4K Gaming Monitor', 1, 449.99),
(10, 'Smartphone X', 1, 899.99);

-- Insert sample activity logs for performance testing
INSERT IGNORE INTO activity_logs (user_id, action, details, ip_address, user_agent, session_id) VALUES
(1, 'login', '{"method": "password", "success": true}', '192.168.1.100', 'Mozilla/5.0 Chrome/91.0', 'sess_001'),
(1, 'view_product', '{"product_id": 1, "product_name": "Gaming Laptop Pro"}', '192.168.1.100', 'Mozilla/5.0 Chrome/91.0', 'sess_001'),
(1, 'add_to_cart', '{"product_id": 1, "quantity": 1}', '192.168.1.100', 'Mozilla/5.0 Chrome/91.0', 'sess_001'),
(1, 'checkout', '{"order_id": 1, "total": 1299.99}', '192.168.1.100', 'Mozilla/5.0 Chrome/91.0', 'sess_001'),
(2, 'login', '{"method": "password", "success": true}', '192.168.1.101', 'Mozilla/5.0 Firefox/89.0', 'sess_002'),
(2, 'search', '{"query": "smartphone", "results": 5}', '192.168.1.101', 'Mozilla/5.0 Firefox/89.0', 'sess_002'),
(2, 'view_product', '{"product_id": 2, "product_name": "Smartphone X"}', '192.168.1.101', 'Mozilla/5.0 Firefox/89.0', 'sess_002'),
(3, 'register', '{"method": "email", "success": true}', '192.168.1.102', 'Mozilla/5.0 Safari/14.0', 'sess_003'),
(3, 'profile_update', '{"fields": ["first_name", "last_name"]}', '192.168.1.102', 'Mozilla/5.0 Safari/14.0', 'sess_003'),
(4, 'password_reset', '{"method": "email", "success": true}', '192.168.1.103', 'Mozilla/5.0 Edge/91.0', 'sess_004');

-- Create some stored procedures for complex testing
DELIMITER $$

-- Procedure to generate test data
CREATE PROCEDURE IF NOT EXISTS GenerateTestUsers(IN num_users INT)
BEGIN
    DECLARE i INT DEFAULT 1;
    DECLARE random_age INT;
    DECLARE random_salary DECIMAL(10,2);

    WHILE i <= num_users DO
        SET random_age = FLOOR(18 + RAND() * 47); -- Age between 18-65
        SET random_salary = ROUND(30000 + RAND() * 120000, 2); -- Salary between 30k-150k

        INSERT IGNORE INTO users (
            username,
            email,
            first_name,
            last_name,
            age,
            salary
        ) VALUES (
            CONCAT('user_', i, '_', FLOOR(RAND() * 10000)),
            CONCAT('user', i, '_', FLOOR(RAND() * 10000), '@testdomain.com'),
            CONCAT('FirstName', i),
            CONCAT('LastName', i),
            random_age,
            random_salary
        );

        SET i = i + 1;
    END WHILE;
END$$

-- Function to calculate user lifetime value
CREATE FUNCTION IF NOT EXISTS CalculateUserLTV(user_id INT)
RETURNS DECIMAL(12,2)
READS SQL DATA
DETERMINISTIC
BEGIN
    DECLARE ltv DECIMAL(12,2) DEFAULT 0;

    SELECT COALESCE(SUM(total_amount), 0) INTO ltv
    FROM orders
    WHERE orders.user_id = user_id AND status != 'cancelled';

    RETURN ltv;
END$$

-- Procedure for complex reporting
CREATE PROCEDURE IF NOT EXISTS GetUserOrderReport(
    IN start_date DATE,
    IN end_date DATE,
    IN min_order_value DECIMAL(10,2)
)
BEGIN
    SELECT
        u.id,
        u.username,
        u.email,
        u.first_name,
        u.last_name,
        COUNT(o.id) as order_count,
        SUM(o.total_amount) as total_spent,
        AVG(o.total_amount) as avg_order_value,
        MIN(o.order_date) as first_order,
        MAX(o.order_date) as last_order,
        CalculateUserLTV(u.id) as lifetime_value
    FROM users u
    LEFT JOIN orders o ON u.id = o.user_id
    WHERE (o.order_date BETWEEN start_date AND end_date)
      AND (o.total_amount >= min_order_value OR o.total_amount IS NULL)
      AND (o.status != 'cancelled' OR o.status IS NULL)
    GROUP BY u.id, u.username, u.email, u.first_name, u.last_name
    HAVING order_count > 0
    ORDER BY total_spent DESC;
END$$

DELIMITER ;

-- Create indexes for better performance during stress testing
ALTER TABLE users ADD INDEX idx_age_salary (age, salary);
ALTER TABLE orders ADD INDEX idx_amount_date (total_amount, order_date);
ALTER TABLE products ADD INDEX idx_category_price (category, price);
ALTER TABLE activity_logs ADD INDEX idx_user_action_date (user_id, action, created_at);

-- Insert additional test data using the stored procedure
-- CALL GenerateTestUsers(100); -- Uncomment to generate more test users

-- Create a table for high-volume testing
CREATE TABLE IF NOT EXISTS performance_test (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    test_name VARCHAR(100) NOT NULL,
    test_value DECIMAL(15,4),
    test_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    test_data JSON,
    category ENUM('read', 'write', 'update', 'delete') DEFAULT 'read',
    INDEX idx_test_name (test_name),
    INDEX idx_test_date (test_date),
    INDEX idx_category (category),
    INDEX idx_composite (test_name, category, test_date)
);

-- Verify table creation
SELECT
    TABLE_NAME as 'Table',
    TABLE_ROWS as 'Rows',
    CREATE_TIME as 'Created'
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = 'testdb'
ORDER BY TABLE_NAME;

-- Show created indexes
SELECT
    TABLE_NAME as 'Table',
    INDEX_NAME as 'Index',
    COLUMN_NAME as 'Column'
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = 'testdb'
ORDER BY TABLE_NAME, INDEX_NAME;

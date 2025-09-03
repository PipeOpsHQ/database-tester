# Database Stress Test Service

A comprehensive multi-database stress testing service built for PipeOps TCP/UDP port testing. This Go service can connect to MySQL, PostgreSQL, MongoDB, Redis, and Microsoft SQL Server databases, run complex stress tests, and generate beautiful HTML reports.

## 🚀 Features

- **Multi-Database Support**: MySQL, PostgreSQL, MongoDB, Redis, Microsoft SQL Server
- **Stress Testing**: Concurrent operations with configurable parameters
- **Complex Queries**: Advanced queries designed to stress test database performance
- **Real-time Reporting**: Live HTML reports with interactive dashboards
- **Dynamic Configuration**: Web UI for easy database credential management
- **RESTful API**: Complete API for programmatic access
- **Connection Monitoring**: Real-time connection status tracking
- **Benchmarking**: Custom query benchmarking capabilities
- **Responsive UI**: Modern, mobile-friendly web interface

## 📋 Prerequisites

- Go 1.21 or later
- Access to one or more of the supported databases
- Network connectivity to database servers

### Supported Databases

| Database             | Version | Driver                             |
| -------------------- | ------- | ---------------------------------- |
| MySQL                | 5.7+    | `github.com/go-sql-driver/mysql`   |
| PostgreSQL           | 10+     | `github.com/lib/pq`                |
| MongoDB              | 4.0+    | `go.mongodb.org/mongo-driver`      |
| Redis                | 5.0+    | `github.com/go-redis/redis/v8`     |
| Microsoft SQL Server | 2017+   | `github.com/denisenkom/go-mssqldb` |

## 🔧 Installation

### 1. Clone the Repository

```bash
git clone <repository-url>
cd tcp-test
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Configure Environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` with your database connection details:

```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# MySQL Configuration
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USERNAME=root
MYSQL_PASSWORD=password
MYSQL_DATABASE=testdb

# PostgreSQL Configuration
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USERNAME=postgres
POSTGRES_PASSWORD=password
POSTGRES_DATABASE=testdb

# MongoDB Configuration
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=testdb

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Microsoft SQL Server Configuration
MSSQL_HOST=localhost
MSSQL_PORT=1433
MSSQL_USERNAME=sa
MSSQL_PASSWORD=Password123!
MSSQL_DATABASE=testdb
```

### 4. Build and Run

```bash
go build -o stress-test ./main.go
./stress-test
```

Or run directly:

```bash
go run main.go
```

## 🖥️ Usage

### Web Interface

Once the service is running, access the web interface:

- **Dashboard**: http://localhost:8080/
- **Configuration UI**: http://localhost:8080/config-ui
- **Full Report**: http://localhost:8080/report
- **Live Report**: http://localhost:8080/live
- **Health Check**: http://localhost:8080/api/health

### Quick Start

1. Open the dashboard at http://localhost:8080/
2. Click "Configure Databases" to set up your database credentials via the web UI
3. Test each database connection using the configuration interface
4. Click "Connect All DBs" to establish database connections
5. Click "Run Stress Test" to start testing all connected databases
6. View results in the "Full Report" page

## 📡 API Reference

### Health Check

```http
GET /api/health
```

Returns service health status and database connectivity.

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2023-12-01T10:00:00Z",
  "version": "1.0.0",
  "databases": {
    "MySQL": "Connected",
    "PostgreSQL": "Connected",
    "MongoDB": "Connected",
    "Redis": "Connected",
    "MSSQL": "Disconnected"
  }
}
```

### Connection Status

```http
GET /api/status
```

Returns detailed connection status for all databases.

### Database Configuration

```http
GET /api/config
```

Returns current database configuration (with masked passwords).

```http
POST /api/config
Content-Type: application/json

{
  "mysql": {
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "newpassword",
    "database": "testdb"
  },
  "postgresql": {
    "host": "localhost",
    "port": 5432,
    "username": "postgres",
    "password": "newpassword",
    "database": "testdb",
    "sslmode": "disable"
  }
}
```

Updates database configuration dynamically without restart.

```http
POST /api/config/test
Content-Type: application/json

{
  "database": "mysql",
  "config": {
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "password",
    "database": "testdb"
  }
}
```

Tests a database configuration before applying.

### Connect to All Databases

```http
POST /api/connect
```

Attempts to connect to all configured databases.

### Run Stress Test

```http
POST /api/test
Content-Type: application/json

{
  "duration": 30,
  "concurrency": 10,
  "query_type": "simple",
  "delay_between": 100,
  "operations_limit": -1
}
```

**Parameters:**

- `duration`: Test duration in seconds (default: 30)
- `concurrency`: Number of concurrent operations (default: 10)
- `query_type`: "simple" or "complex" (default: "simple")
- `delay_between`: Delay between operations in milliseconds (default: 100)
- `operations_limit`: Maximum operations per worker, -1 for unlimited (default: -1)

### Test Specific Database

```http
POST /api/test/{database}
Content-Type: application/json

{
  "duration": 60,
  "concurrency": 20,
  "query_type": "complex"
}
```

Replace `{database}` with one of: `MySQL`, `PostgreSQL`, `MongoDB`, `Redis`, `MSSQL`

### Custom Benchmark

```http
POST /api/benchmark
Content-Type: application/json

{
  "database": "MySQL",
  "query": "SELECT COUNT(*) FROM information_schema.tables",
  "iterations": 100
}
```

### Get Latest Results

```http
GET /api/results
```

Returns the latest stress test results in JSON format.

## 🐳 Docker Support

### Using Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o stress-test ./main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/stress-test .
COPY --from=builder /app/templates ./templates

EXPOSE 8080
CMD ["./stress-test"]
```

Build and run:

```bash
docker build -t db-stress-test .
docker run -p 8080:8080 --env-file .env db-stress-test
```

### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: "3.8"

services:
  stress-test:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SERVER_HOST=0.0.0.0
      - SERVER_PORT=8080
      - MYSQL_HOST=mysql
      - POSTGRES_HOST=postgres
      - MONGO_URI=mongodb://mongo:27017
      - REDIS_HOST=redis
    depends_on:
      - mysql
      - postgres
      - mongo
      - redis

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: password
      MYSQL_DATABASE: testdb
    ports:
      - "3306:3306"

  postgres:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: password
      POSTGRES_DB: testdb
    ports:
      - "5432:5432"

  mongo:
    image: mongo:7
    ports:
      - "27017:27017"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

Run with Docker Compose:

```bash
docker-compose up -d
```

## 🔍 Monitoring and Metrics

The service provides comprehensive metrics for each database:

### Connection Metrics

- Connection status (Connected/Disconnected)
- Connection establishment time
- Connection health checks

### Performance Metrics

- Operations per second
- Average latency
- 95th and 99th percentile latencies
- Min/Max response times

### Reliability Metrics

- Success rate
- Error rate
- Error details and categorization
- Total operations completed

## 🛠️ Configuration Options

## 🎛️ Configuration Management

### Web-Based Configuration UI

Access the configuration interface at `http://localhost:8080/config-ui` to:

- **Add Database Credentials**: Enter connection details through intuitive forms
- **Test Connections**: Verify connectivity before saving configurations
- **Update Settings**: Modify database parameters without service restart
- **Real-time Status**: See connection status updates instantly

### Dynamic Configuration

The service supports runtime configuration updates through:

- Web UI forms for each database type
- REST API endpoints for programmatic updates
- Automatic reconnection when credentials change
- Configuration validation and testing

### Environment Variables

| Variable            | Description            | Default                     |
| ------------------- | ---------------------- | --------------------------- |
| `SERVER_HOST`       | Server bind address    | `0.0.0.0`                   |
| `SERVER_PORT`       | Server port            | `8080`                      |
| `MYSQL_HOST`        | MySQL hostname         | `localhost`                 |
| `MYSQL_PORT`        | MySQL port             | `3306`                      |
| `MYSQL_USERNAME`    | MySQL username         | `root`                      |
| `MYSQL_PASSWORD`    | MySQL password         | `password`                  |
| `MYSQL_DATABASE`    | MySQL database         | `testdb`                    |
| `POSTGRES_HOST`     | PostgreSQL hostname    | `localhost`                 |
| `POSTGRES_PORT`     | PostgreSQL port        | `5432`                      |
| `POSTGRES_USERNAME` | PostgreSQL username    | `postgres`                  |
| `POSTGRES_PASSWORD` | PostgreSQL password    | `password`                  |
| `POSTGRES_DATABASE` | PostgreSQL database    | `testdb`                    |
| `POSTGRES_SSLMODE`  | PostgreSQL SSL mode    | `disable`                   |
| `MONGO_URI`         | MongoDB connection URI | `mongodb://localhost:27017` |
| `MONGO_DATABASE`    | MongoDB database       | `testdb`                    |
| `REDIS_HOST`        | Redis hostname         | `localhost`                 |
| `REDIS_PORT`        | Redis port             | `6379`                      |
| `REDIS_PASSWORD`    | Redis password         | (empty)                     |
| `REDIS_DB`          | Redis database number  | `0`                         |
| `MSSQL_HOST`        | MSSQL hostname         | `localhost`                 |
| `MSSQL_PORT`        | MSSQL port             | `1433`                      |
| `MSSQL_USERNAME`    | MSSQL username         | `sa`                        |
| `MSSQL_PASSWORD`    | MSSQL password         | `Password123!`              |
| `MSSQL_DATABASE`    | MSSQL database         | `testdb`                    |

## 🚨 Troubleshooting

### Common Issues

#### Database Connection Failures

**Problem**: Database shows as "Disconnected" in status

**Solutions:**

1. Check database server is running
2. Verify connection parameters in `.env`
3. Check network connectivity
4. Verify database credentials
5. Check firewall settings

#### Permission Errors

**Problem**: "Access denied" or "Authentication failed"

**Solutions:**

1. Verify username and password
2. Check user permissions in database
3. Ensure user has necessary privileges for test operations

#### Port Binding Issues

**Problem**: "Address already in use" error

**Solutions:**

1. Change `SERVER_PORT` in environment variables
2. Kill existing processes using the port: `lsof -ti:8080 | xargs kill`

#### Memory Issues

**Problem**: High memory usage during stress tests

**Solutions:**

1. Reduce concurrency level
2. Add delays between operations
3. Limit operations per test
4. Increase available system memory

### Debug Mode

Enable verbose logging by setting log level:

```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

### Performance Tuning

For optimal performance:

1. **Connection Pooling**: Adjust `MaxOpenConns` and `MaxIdleConns` in database configurations
2. **Concurrency**: Start with low concurrency (5-10) and gradually increase
3. **Test Duration**: Use shorter durations for initial testing
4. **Query Complexity**: Start with simple queries before moving to complex ones

## 📊 Understanding Reports

### Dashboard Metrics

- **Total Databases**: Number of configured databases
- **Connected**: Number of successfully connected databases
- **Total Operations**: Sum of all operations across databases
- **Success Rate**: Percentage of successful operations
- **Avg Ops/Sec**: Average operations per second across all databases
- **Fastest DB**: Database with highest operations per second

### Individual Database Metrics

- **Operations**: Total operations performed
- **Success/Failed**: Breakdown of operation results
- **Ops/Sec**: Operations per second for this database
- **Error Rate**: Percentage of failed operations
- **Latency Chart**: Visual representation of response times

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and add tests
4. Commit your changes: `git commit -am 'Add feature'`
5. Push to the branch: `git push origin feature-name`
6. Submit a pull request

### Development Setup

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests
go test ./...

# Run linter
golangci-lint run

# Format code
go fmt ./...
```

## 🎯 Configuration UI Features

### Easy Database Setup

- **Visual Forms**: Intuitive web forms for each database type
- **Real-time Testing**: Test connections before applying changes
- **Status Indicators**: Visual feedback on connection status
- **Auto-completion**: Smart defaults for common configurations

### PipeOps Integration

- **Quick Setup**: Easily add PipeOps-exposed database endpoints
- **Credential Management**: Secure handling of database credentials
- **Connection Validation**: Verify exposed ports are accessible
- **Bulk Operations**: Test and configure multiple databases at once

### Example: Adding PipeOps MySQL

1. Open Configuration UI: `http://localhost:8080/config-ui`
2. Navigate to MySQL section
3. Enter your PipeOps details:
   - Host: `your-mysql-host.pipeops.io`
   - Port: `3306`
   - Username: `your_username`
   - Password: `your_password`
   - Database: `your_database`
4. Click "Test Connection" to verify
5. Click "Save Config" to apply changes
6. Start stress testing immediately

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

- Create an issue for bug reports or feature requests
- Check existing issues before creating new ones
- Provide detailed information including environment details and error messages

## 🏗️ Architecture

```
tcp-test/
├── config/          # Configuration management
├── connectors/      # Database connectors and interfaces
├── stresstester/    # Stress testing logic
├── report/          # HTML report generation
├── server/          # HTTP server and API handlers
├── templates/       # HTML templates
├── main.go          # Application entry point
└── README.md        # This file
```

## 🔮 Roadmap

- [ ] Add support for additional databases (ClickHouse, CouchDB, etc.)
- [ ] Implement test scheduling and automation
- [ ] Add Prometheus metrics export
- [ ] Create Grafana dashboard templates
- [ ] Add email notifications for test results
- [ ] Implement test result history and trends
- [ ] Add load balancer health check endpoints
- [ ] Create Kubernetes deployment manifests

---

Built with ❤️ for PipeOps TCP/UDP port testing
# database-tester

# Changelog

All notable changes to the Database Stress Test Service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2025-09-03

### Fixed
- **Critical JSON Parsing Error**: Fixed "JSON.parse: unexpected character at line 1 column 1" error in configuration UI
  - Added proper Content-Type header validation before parsing JSON responses
  - Implemented graceful error handling for non-JSON server responses
  - Added comprehensive error checking in all AJAX calls
  - Enhanced error messages to provide more context

### Added
- **Server-side Improvements**:
  - Added `writeJSONResponse()` helper method for consistent JSON responses
  - Added `writeJSONError()` helper method for standardized error responses
  - Implemented panic recovery middleware for all configuration handlers
  - Added comprehensive error logging throughout the application

### Changed
- **Enhanced Error Handling**:
  - All API endpoints now consistently return JSON responses with proper Content-Type headers
  - Improved error messages to include error types for better debugging
  - Configuration update responses now include `success` field for better client-side handling
  - Added response status code consistency across all endpoints

### Technical Details
- **Frontend Changes**:
  - Modified `saveConfig()` function to validate response Content-Type before JSON parsing
  - Updated `saveAllConfigs()` function with proper error handling
  - Enhanced `testAllConnections()` function to handle non-JSON responses
  - Added fallback error handling for all fetch operations

- **Backend Changes**:
  - Updated `updateConfigHandler()` with panic recovery and consistent JSON responses
  - Enhanced `testConfigHandler()` with better error handling
  - Improved `connectHandler()` to use new JSON response helpers
  - Added comprehensive logging for debugging purposes

### Security
- Enhanced input validation and error handling to prevent information leakage
- Added proper error type categorization for security analysis

## [1.0.0] - 2025-09-03

### Added
- Initial release of Database Stress Test Service
- Multi-database support: MySQL, PostgreSQL, MongoDB, Redis, Microsoft SQL Server
- Web-based configuration UI for database credentials
- RESTful API for programmatic access
- Real-time connection status monitoring
- Comprehensive stress testing capabilities
- HTML report generation
- Docker support with docker-compose configuration

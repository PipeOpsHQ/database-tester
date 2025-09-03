#!/bin/bash

# Database Stress Test Service - Test Script
# This script verifies the basic functionality of the service

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

# Service configuration
SERVICE_HOST="localhost"
SERVICE_PORT="8080"
BASE_URL="http://${SERVICE_HOST}:${SERVICE_PORT}"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

echo -e "${CYAN}========================================${NC}"
echo -e "${WHITE}Database Stress Test Service - Test Suite${NC}"
echo -e "${CYAN}========================================${NC}"
echo -e "${BLUE}Testing service at: ${BASE_URL}${NC}"
echo ""

# Helper functions
print_test_header() {
    echo -e "${PURPLE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓ PASS${NC} - $1"
    ((TESTS_PASSED++))
}

print_failure() {
    echo -e "${RED}✗ FAIL${NC} - $1"
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${BLUE}ℹ INFO${NC} - $1"
}

print_warning() {
    echo -e "${YELLOW}⚠ WARN${NC} - $1"
}

# Test HTTP endpoint
test_http_endpoint() {
    local endpoint="$1"
    local description="$2"
    local expected_status="${3:-200}"

    print_test_header "Testing ${description}"

    # Check if endpoint is reachable
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null)
    local curl_exit_code=$?

    if [ $curl_exit_code -ne 0 ]; then
        print_failure "${description} - Service unreachable (curl exit code: $curl_exit_code)"
        return 1
    fi

    if [ "$response_code" = "$expected_status" ]; then
        print_success "${description} - HTTP $response_code"
        return 0
    else
        print_failure "${description} - Expected HTTP $expected_status, got $response_code"
        return 1
    fi
}

# Test JSON API endpoint
test_json_endpoint() {
    local endpoint="$1"
    local description="$2"
    local method="${3:-GET}"
    local data="$4"

    print_test_header "Testing ${description}"

    local curl_cmd="curl -s"
    if [ "$method" = "POST" ]; then
        curl_cmd="$curl_cmd -X POST -H 'Content-Type: application/json'"
        if [ -n "$data" ]; then
            curl_cmd="$curl_cmd -d '$data'"
        fi
    fi

    local response=$(eval "$curl_cmd ${BASE_URL}${endpoint}" 2>/dev/null)
    local curl_exit_code=$?

    if [ $curl_exit_code -ne 0 ]; then
        print_failure "${description} - Service unreachable"
        return 1
    fi

    # Check if response is valid JSON
    if echo "$response" | jq . >/dev/null 2>&1; then
        print_success "${description} - Valid JSON response"
        # Print first few fields of response
        echo "$response" | jq -r 'keys[0:3] | join(", ")' 2>/dev/null | sed 's/^/    Response fields: /'
        return 0
    else
        print_failure "${description} - Invalid JSON response"
        echo "    Response: ${response:0:100}..."
        return 1
    fi
}

# Check if service is running
check_service_status() {
    print_test_header "Checking if service is running"

    # Try to connect to the port
    if timeout 3 bash -c "</dev/tcp/${SERVICE_HOST}/${SERVICE_PORT}" 2>/dev/null; then
        print_success "Service is listening on port ${SERVICE_PORT}"
        return 0
    else
        print_failure "Service is not running or not accessible on port ${SERVICE_PORT}"
        print_info "Make sure to start the service with: ./stress-test or make run"
        return 1
    fi
}

# Run comprehensive tests
run_tests() {
    echo -e "${YELLOW}Starting test suite...${NC}"
    echo ""

    # 1. Check if service is running
    if ! check_service_status; then
        echo -e "\n${RED}Cannot continue testing - service is not running${NC}"
        exit 1
    fi
    echo ""

    # 2. Test basic HTTP endpoints
    test_http_endpoint "/" "Dashboard page"
    test_http_endpoint "/config-ui" "Configuration UI page"
    test_http_endpoint "/report" "Report page" "200"
    test_http_endpoint "/live" "Live report page"
    echo ""

    # 3. Test API endpoints
    test_json_endpoint "/api/health" "Health check API"
    test_json_endpoint "/api/status" "Status API"
    test_json_endpoint "/api/config" "Configuration API"
    echo ""

    # 4. Test POST endpoints
    print_test_header "Testing database connection"
    test_json_endpoint "/api/connect" "Database connection API" "POST"
    echo ""

    sleep 2  # Give connections time to establish

    # 5. Test stress test endpoint (short duration)
    print_test_header "Testing stress test functionality"
    local test_config='{"duration": 5, "concurrency": 2, "query_type": "simple"}'
    test_json_endpoint "/api/test" "Stress test API" "POST" "$test_config"
    echo ""

    # 6. Test benchmark endpoint
    print_test_header "Testing benchmark functionality"
    local benchmark_config='{"database": "MySQL", "query": "SELECT 1", "iterations": 5}'
    test_json_endpoint "/api/benchmark" "Benchmark API" "POST" "$benchmark_config"
    echo ""

    # 7. Test results endpoint
    test_json_endpoint "/api/results" "Results API"
    echo ""

    # 8. Test configuration management
    print_test_header "Testing configuration management"
    test_config_update
    echo ""

    # 9. Test MongoDB functionality
    print_test_header "Testing MongoDB integration"
    test_mongodb_functionality
    echo ""
}

# Performance test
run_performance_test() {
    print_test_header "Running performance validation"

    echo -e "${BLUE}Testing concurrent connections...${NC}"

    # Test multiple concurrent requests
    local start_time=$(date +%s)
    for i in {1..5}; do
        curl -s "${BASE_URL}/api/health" >/dev/null &
    done
    wait
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    if [ $duration -le 3 ]; then
        print_success "Concurrent requests handled efficiently (${duration}s)"
    else
        print_warning "Concurrent requests took longer than expected (${duration}s)"
    fi
}

# Test configuration update functionality
test_config_update() {
    print_test_header "Testing configuration update API"

    # Test configuration retrieval
    local config_response=$(curl -s "${BASE_URL}/api/config" 2>/dev/null)
    local curl_exit_code=$?

    if [ $curl_exit_code -ne 0 ]; then
        print_failure "Configuration API - Service unreachable"
        return 1
    fi

    if echo "$config_response" | jq . >/dev/null 2>&1; then
        print_success "Configuration API - Valid JSON response"
    else
        print_failure "Configuration API - Invalid JSON response"
        return 1
    fi

    # Test configuration update with sample MySQL config
    print_test_header "Testing configuration update"
    local update_config='{"mysql": {"host": "test-host", "port": 3306, "username": "test-user", "password": "test-pass", "database": "test-db"}}'

    local update_response=$(curl -s -X POST "${BASE_URL}/api/config" \
        -H "Content-Type: application/json" \
        -d "$update_config" 2>/dev/null)
    local update_exit_code=$?

    if [ $update_exit_code -ne 0 ]; then
        print_failure "Configuration update - Service unreachable"
        return 1
    fi

    if echo "$update_response" | jq . >/dev/null 2>&1; then
        print_success "Configuration update - Valid JSON response"

        # Check if update was successful
        if echo "$update_response" | jq -e '.updated == true' >/dev/null 2>&1; then
            print_success "Configuration update - Update confirmed"
        else
            print_warning "Configuration update - Update status unclear"
        fi
    else
        print_failure "Configuration update - Invalid JSON response"
        return 1
    fi

    return 0
}

# Test MongoDB functionality
test_mongodb_functionality() {
    print_test_header "Testing MongoDB configuration and connectivity"

    # Test MongoDB configuration update
    local mongodb_config='{"mongodb": {"uri": "mongodb://test:test@localhost:27017", "database": "testdb"}}'

    local response=$(curl -s -X POST "${BASE_URL}/api/config" \
        -H "Content-Type: application/json" \
        -d "$mongodb_config" 2>/dev/null)
    local curl_exit_code=$?

    if [ $curl_exit_code -eq 0 ]; then
        print_success "MongoDB configuration update - Request successful"
    else
        print_failure "MongoDB configuration update - Request failed"
        return 1
    fi

    # Test MongoDB connection test endpoint
    local test_config='{"database": "mongodb", "config": {"uri": "mongodb://localhost:27017", "database": "testdb"}}'

    local test_response=$(curl -s -X POST "${BASE_URL}/api/config/test" \
        -H "Content-Type: application/json" \
        -d "$test_config" 2>/dev/null)
    local test_exit_code=$?

    if [ $test_exit_code -eq 0 ]; then
        print_success "MongoDB connection test - API accessible"
    else
        print_failure "MongoDB connection test - API not accessible"
    fi

    return 0
}

# Service information
show_service_info() {
    echo -e "\n${CYAN}Service Information:${NC}"
    echo -e "${WHITE}Dashboard:${NC}    ${BASE_URL}/"
    echo -e "${WHITE}Health:${NC}       ${BASE_URL}/api/health"
    echo -e "${WHITE}Status:${NC}       ${BASE_URL}/api/status"
    echo -e "${WHITE}Report:${NC}       ${BASE_URL}/report"
    echo -e "${WHITE}Live Report:${NC}  ${BASE_URL}/live"
    echo ""

    echo -e "${CYAN}Quick Commands:${NC}"
    echo -e "${WHITE}Health Check:${NC}  curl ${BASE_URL}/api/health"
    echo -e "${WHITE}Connect DBs:${NC}   curl -X POST ${BASE_URL}/api/connect"
    echo -e "${WHITE}Run Test:${NC}      curl -X POST ${BASE_URL}/api/test -H 'Content-Type: application/json' -d '{\"duration\": 10, \"concurrency\": 5}'"
    echo -e "${WHITE}Config UI:${NC}     ${BASE_URL}/config-ui"
    echo -e "${WHITE}Get Config:${NC}    curl ${BASE_URL}/api/config"
    echo -e "${WHITE}MongoDB Test:${NC}  curl -X POST ${BASE_URL}/api/config/test -d '{\"database\":\"mongodb\",\"config\":{\"uri\":\"mongodb://localhost:27017\",\"database\":\"testdb\"}}'"
}

# Main execution
main() {
    # Check if required tools are available
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}Error: curl is required but not installed${NC}"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        print_warning "jq not found - JSON parsing will be limited"
    fi

    # Run the tests
    run_tests

    # Performance test
    run_performance_test

    # Show summary
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${WHITE}Test Summary${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo -e "${GREEN}Tests Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Tests Failed: ${TESTS_FAILED}${NC}"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}🎉 All tests passed! Service is working correctly.${NC}"
    else
        echo -e "\n${YELLOW}⚠ Some tests failed. Check the output above for details.${NC}"
    fi

    # Show service information
    show_service_info

    echo -e "\n${BLUE}For detailed testing, visit the dashboard at: ${BASE_URL}${NC}"

    # Exit with appropriate code
    if [ $TESTS_FAILED -eq 0 ]; then
        exit 0
    else
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Database Stress Test Service - Test Script"
        echo ""
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --info, -i     Show service information only"
        echo "  --quick, -q    Run quick tests only"
        echo "  --config, -c   Test configuration management only"
        echo "  --mongodb, -m  Test MongoDB functionality only"
        echo ""
        echo "The script will test various endpoints and functionality of the service."
        echo "Make sure the service is running before executing this script."
        exit 0
        ;;
    --info|-i)
        show_service_info
        exit 0
        ;;
    --quick|-q)
        check_service_status
        test_http_endpoint "/" "Dashboard"
        test_json_endpoint "/api/health" "Health API"
        exit 0
        ;;
    --config|-c)
        check_service_status
        test_http_endpoint "/config-ui" "Configuration UI"
        test_json_endpoint "/api/config" "Configuration API"
        test_config_update
        exit 0
        ;;
    --mongodb|-m)
        check_service_status
        test_mongodb_functionality
        exit 0
        ;;
    *)
        main
        ;;
esac

#!/usr/bin/env bash

# Test script for all madock platforms
# Creates test projects, runs basic commands, and cleans up

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MADOCK_DIR="$(dirname "$SCRIPT_DIR")"
TMP_DIR="$MADOCK_DIR/tmp"
MADOCK_BIN="$MADOCK_DIR/madock"

# Test results (using simple variables instead of associative array for compatibility)
RESULTS_FILE="/tmp/madock-test-results.txt"
FAILED_TESTS=0
PASSED_TESTS=0

# Platforms to test
PLATFORMS="magento2 shopware prestashop pwa shopify custom"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_header() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}========================================${NC}"
    echo ""
}

# Check if madock binary exists
check_madock() {
    if [[ ! -f "$MADOCK_BIN" ]]; then
        log_error "Madock binary not found at $MADOCK_BIN"
        log_info "Building madock..."
        cd "$MADOCK_DIR"
        go build -o madock
        if [[ ! -f "$MADOCK_BIN" ]]; then
            log_error "Failed to build madock"
            exit 1
        fi
        log_success "Madock built successfully"
    fi
}

# Create tmp directory
setup_tmp_dir() {
    log_info "Setting up tmp directory at $TMP_DIR"
    mkdir -p "$TMP_DIR"
}

# Clean up a test project
cleanup_project() {
    local project_name=$1
    local project_dir="$TMP_DIR/$project_name"

    log_info "Cleaning up project: $project_name"

    # Stop containers if running
    if [[ -d "$project_dir" ]]; then
        cd "$project_dir"
        "$MADOCK_BIN" stop 2>/dev/null || true
    fi

    # Remove project directories
    rm -rf "$project_dir"
    rm -rf "$MADOCK_DIR/projects/$project_name"
    rm -rf "$MADOCK_DIR/aruntime/projects/$project_name"

    # Remove from cache
    rm -rf "$MADOCK_DIR/cache/$project_name-proxy.conf"

    log_success "Cleaned up $project_name"
}

# Get platform-specific setup arguments
get_platform_args() {
    local platform=$1
    local project_name=$2
    local host="test-${platform}.test"

    case $platform in
        magento2)
            echo "--platform=magento2 --hosts=${host}:base --php=8.3 --db=10.6 --search-engine=OpenSearch --search-engine-version=2.12.0 --redis=7.2 --composer=2"
            ;;
        shopware)
            echo "--platform=shopware --hosts=${host}:base --php=8.2 --db=10.6 --composer=2"
            ;;
        prestashop)
            echo "--platform=prestashop --hosts=${host}:base --php=8.1 --db=10.6 --composer=2"
            ;;
        pwa)
            echo "--platform=pwa --hosts=${host}:base --nodejs=20.19.0 --yarn=1.22.19 --pwa-backend-url=https://magento.test"
            ;;
        shopify)
            echo "--platform=shopify --hosts=${host}:base --nodejs=20.19.0"
            ;;
        custom)
            # Custom platform for Laravel
            echo "--platform=custom --hosts=${host}:base --php=8.3 --db=10.6 --composer=2"
            ;;
        *)
            echo "--platform=$platform --hosts=${host}:base"
            ;;
    esac
}

# Test a single platform
test_platform() {
    local platform=$1
    local project_name="test-${platform}"
    local project_dir="$TMP_DIR/$project_name"
    local host="test-${platform}.test"
    local test_passed=true

    log_header "Testing platform: $platform"

    # Cleanup any previous test
    cleanup_project "$project_name"

    # Create project directory
    log_info "Creating project directory: $project_dir"
    mkdir -p "$project_dir"
    cd "$project_dir"

    # For custom platform, initialize Laravel structure
    if [[ "$platform" == "custom" ]]; then
        log_info "Initializing Laravel-like structure for custom platform"
        mkdir -p public
        echo '<?php echo "Hello from Laravel!";' > public/index.php
        cat > composer.json << 'EOF'
{
    "name": "test/laravel-app",
    "type": "project",
    "require": {
        "php": "^8.1"
    }
}
EOF
    fi

    # Run setup
    log_info "Running madock setup for $platform..."
    local setup_args=$(get_platform_args "$platform" "$project_name")

    if ! "$MADOCK_BIN" setup $setup_args 2>&1 | tee /tmp/madock-setup-$platform.log; then
        log_error "Setup failed for $platform"
        test_passed=false
    else
        log_success "Setup completed for $platform"
    fi

    # Check if config was created
    if [[ -f "$MADOCK_DIR/projects/$project_name/config.xml" ]]; then
        log_success "Config file created"
    else
        log_error "Config file not found"
        test_passed=false
    fi

    # Test start command (but don't actually start to save resources)
    log_info "Testing status command..."
    if "$MADOCK_BIN" status 2>&1; then
        log_success "Status command works"
    else
        log_warning "Status command returned non-zero (expected if containers not running)"
    fi

    # Test info command
    log_info "Testing info command..."
    if "$MADOCK_BIN" info 2>&1; then
        log_success "Info command works"
    else
        log_error "Info command failed"
        test_passed=false
    fi

    # Test config:list command
    log_info "Testing config:list command..."
    if "$MADOCK_BIN" config:list 2>&1 | head -20; then
        log_success "Config:list command works"
    else
        log_error "Config:list command failed"
        test_passed=false
    fi

    # Cleanup
    cleanup_project "$project_name"

    # Record result
    if $test_passed; then
        echo "$platform=PASSED" >> "$RESULTS_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "Platform $platform: ALL TESTS PASSED"
    else
        echo "$platform=FAILED" >> "$RESULTS_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "Platform $platform: SOME TESTS FAILED"
    fi

    echo ""
}

# Test with containers (optional, slower)
test_platform_with_containers() {
    local platform=$1
    local project_name="test-${platform}-full"
    local project_dir="$TMP_DIR/$project_name"

    log_header "Full test with containers: $platform"

    cleanup_project "$project_name"
    mkdir -p "$project_dir"
    cd "$project_dir"

    # Initialize for custom
    if [[ "$platform" == "custom" ]]; then
        mkdir -p public
        echo '<?php phpinfo();' > public/index.php
    fi

    local setup_args=$(get_platform_args "$platform" "$project_name")

    # Setup
    log_info "Running setup..."
    "$MADOCK_BIN" setup $setup_args

    # Start containers
    log_info "Starting containers..."
    "$MADOCK_BIN" start

    # Wait for containers to be ready
    log_info "Waiting for containers to be ready..."
    sleep 10

    # Test status
    log_info "Checking status..."
    "$MADOCK_BIN" status

    # Test bash access
    log_info "Testing bash access..."
    "$MADOCK_BIN" bash -c "php -v" || true

    # Stop containers
    log_info "Stopping containers..."
    "$MADOCK_BIN" stop

    # Cleanup
    cleanup_project "$project_name"

    log_success "Full test completed for $platform"
}

# Print summary
print_summary() {
    log_header "TEST SUMMARY"

    local platform_count=$(echo $PLATFORMS | wc -w | tr -d ' ')
    echo -e "Total platforms tested: $platform_count"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo ""

    echo "Results by platform:"
    for platform in $PLATFORMS; do
        local result="NOT RUN"
        if [[ -f "$RESULTS_FILE" ]]; then
            result=$(grep "^$platform=" "$RESULTS_FILE" 2>/dev/null | cut -d= -f2 || echo "NOT RUN")
        fi
        if [[ "$result" == "PASSED" ]]; then
            echo -e "  ${GREEN}$platform: $result${NC}"
        else
            echo -e "  ${RED}$platform: $result${NC}"
        fi
    done

    echo ""
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "All tests passed!"
        return 0
    else
        log_error "Some tests failed!"
        return 1
    fi
}

# Cleanup all test projects
cleanup_all() {
    log_header "Cleaning up all test projects"

    for platform in $PLATFORMS; do
        cleanup_project "test-${platform}"
        cleanup_project "test-${platform}-full"
    done

    # Remove tmp directory if empty
    if [[ -d "$TMP_DIR" ]] && [[ -z "$(ls -A "$TMP_DIR")" ]]; then
        rmdir "$TMP_DIR"
    fi

    # Remove results file
    rm -f "$RESULTS_FILE"

    log_success "Cleanup completed"
}

# Main
main() {
    local run_full_tests=false
    local specific_platform=""
    local cleanup_only=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --full)
                run_full_tests=true
                shift
                ;;
            --platform)
                specific_platform=$2
                shift 2
                ;;
            --cleanup)
                cleanup_only=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --full          Run full tests with containers (slower)"
                echo "  --platform NAME Test only specific platform"
                echo "  --cleanup       Only cleanup, don't run tests"
                echo "  --help          Show this help"
                echo ""
                echo "Platforms: $PLATFORMS"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    log_header "MADOCK PLATFORM TESTS"
    log_info "Madock directory: $MADOCK_DIR"
    log_info "Tmp directory: $TMP_DIR"

    check_madock
    setup_tmp_dir

    # Initialize results file
    rm -f "$RESULTS_FILE"
    touch "$RESULTS_FILE"

    if $cleanup_only; then
        cleanup_all
        exit 0
    fi

    # Run tests
    if [[ -n "$specific_platform" ]]; then
        if echo "$PLATFORMS" | grep -qw "$specific_platform"; then
            test_platform "$specific_platform"
            if $run_full_tests; then
                test_platform_with_containers "$specific_platform"
            fi
        else
            log_error "Unknown platform: $specific_platform"
            log_info "Available platforms: $PLATFORMS"
            exit 1
        fi
    else
        for platform in $PLATFORMS; do
            test_platform "$platform"
        done

        if $run_full_tests; then
            for platform in $PLATFORMS; do
                test_platform_with_containers "$platform"
            done
        fi
    fi

    print_summary
}

main "$@"

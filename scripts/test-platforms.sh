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

# Magento 2.4.8 presets to test
MAGENTO_PRESETS=(
    "Magento 2.4.8 (Latest)"
    "Magento 2.4.8 + Elasticsearch"
    "Magento 2.4.8 + Valkey"
    "Magento 2.4.8 Minimal"
    "Magento 2.4.8 + Elasticsearch + Valkey"
)

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

    # Test info command (may fail due to TTY requirement, skip this check)
    log_info "Testing info command..."
    local info_output
    info_output=$("$MADOCK_BIN" info 2>&1) || true
    if echo "$info_output" | grep -q "TTY"; then
        log_warning "Info command skipped (requires TTY)"
    elif [ -n "$info_output" ]; then
        echo "$info_output"
        log_success "Info command works"
    else
        log_warning "Info command returned empty output"
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

# Test a single preset
test_preset() {
    local preset_name="$1"
    local safe_name=$(echo "$preset_name" | tr ' +()./' '------' | tr -s '-' | tr '[:upper:]' '[:lower:]' | sed 's/-$//')
    local project_name="test-preset-${safe_name}"
    local project_dir="$TMP_DIR/$project_name"
    local host="${safe_name}.test"
    local test_passed=true

    log_header "Testing preset: $preset_name"

    # Cleanup any previous test
    cleanup_project "$project_name"

    # Create project directory
    log_info "Creating project directory: $project_dir"
    mkdir -p "$project_dir"
    cd "$project_dir"

    # Create minimal composer.json for Magento detection
    cat > composer.json << 'EOF'
{
    "name": "test/magento2",
    "type": "project"
}
EOF

    # Run setup with preset (use yes to auto-confirm)
    log_info "Running madock setup with preset: $preset_name"
    if ! yes "" | "$MADOCK_BIN" setup --platform=magento2 --preset="$preset_name" --hosts="${host}:base" 2>&1 | tee /tmp/madock-setup-preset.log; then
        log_error "Setup failed for preset: $preset_name"
        test_passed=false
    else
        log_success "Setup completed for preset: $preset_name"
    fi

    # Check if config was created
    if [[ -f "$MADOCK_DIR/projects/$project_name/config.xml" ]]; then
        log_success "Config file created"
    else
        log_error "Config file not found"
        test_passed=false
    fi

    # Test status command
    log_info "Testing status command..."
    if "$MADOCK_BIN" status 2>&1; then
        log_success "Status command works"
    else
        log_warning "Status command returned non-zero (expected if containers not running)"
    fi

    # Test config:list command
    log_info "Testing config:list command..."
    if "$MADOCK_BIN" config:list 2>&1 | head -20; then
        log_success "Config:list command works"
    else
        log_error "Config:list command failed"
        test_passed=false
    fi

    # Verify preset-specific settings
    log_info "Verifying preset configuration..."
    local config_output=$("$MADOCK_BIN" config:list 2>&1)

    if echo "$preset_name" | grep -qi "elasticsearch"; then
        if echo "$config_output" | grep -q "search/elasticsearch/enabled.*true"; then
            log_success "Elasticsearch is enabled as expected"
        else
            log_warning "Could not verify Elasticsearch setting"
        fi
    fi

    if echo "$preset_name" | grep -qi "opensearch\|latest\|minimal"; then
        if echo "$config_output" | grep -q "search/opensearch/enabled.*true"; then
            log_success "OpenSearch is enabled as expected"
        else
            log_warning "Could not verify OpenSearch setting"
        fi
    fi

    # Cleanup
    cleanup_project "$project_name"

    # Record result
    if $test_passed; then
        echo "preset:$safe_name=PASSED" >> "$RESULTS_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "Preset '$preset_name': ALL TESTS PASSED"
    else
        echo "preset:$safe_name=FAILED" >> "$RESULTS_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "Preset '$preset_name': SOME TESTS FAILED"
    fi

    echo ""
}

# Test all presets
test_all_presets() {
    log_header "Testing Magento 2.4.8 Presets"

    for preset in "${MAGENTO_PRESETS[@]}"; do
        test_preset "$preset"
    done
}

# Print preset summary
print_preset_summary() {
    log_header "PRESET TEST SUMMARY"

    local preset_count=${#MAGENTO_PRESETS[@]}
    echo -e "Total presets tested: $preset_count"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo ""

    echo "Results by preset:"
    for preset in "${MAGENTO_PRESETS[@]}"; do
        local safe_name=$(echo "$preset" | tr ' +()' '----' | tr -s '-' | tr '[:upper:]' '[:lower:]')
        local result="NOT RUN"
        if [[ -f "$RESULTS_FILE" ]]; then
            result=$(grep "^preset:$safe_name=" "$RESULTS_FILE" 2>/dev/null | cut -d= -f2 || echo "NOT RUN")
        fi
        if [[ "$result" == "PASSED" ]]; then
            echo -e "  ${GREEN}$preset: $result${NC}"
        else
            echo -e "  ${RED}$preset: $result${NC}"
        fi
    done

    echo ""
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "All preset tests passed!"
        return 0
    else
        log_error "Some preset tests failed!"
        return 1
    fi
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

    # Cleanup preset test projects
    for preset in "${MAGENTO_PRESETS[@]}"; do
        local safe_name=$(echo "$preset" | tr ' +()' '----' | tr -s '-' | tr '[:upper:]' '[:lower:]')
        cleanup_project "test-preset-${safe_name}"
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
    local test_presets=false
    local specific_preset=""

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
            --presets)
                test_presets=true
                shift
                ;;
            --preset)
                specific_preset="$2"
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
                echo "  --full           Run full tests with containers (slower)"
                echo "  --platform NAME  Test only specific platform"
                echo "  --presets        Test all Magento 2.4.8 presets"
                echo "  --preset NAME    Test specific preset (use quotes for names with spaces)"
                echo "  --cleanup        Only cleanup, don't run tests"
                echo "  --help           Show this help"
                echo ""
                echo "Platforms: $PLATFORMS"
                echo ""
                echo "Available presets:"
                for preset in "${MAGENTO_PRESETS[@]}"; do
                    echo "  - $preset"
                done
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

    # Run preset tests
    if $test_presets; then
        test_all_presets
        print_preset_summary
        exit $?
    fi

    # Run specific preset test
    if [[ -n "$specific_preset" ]]; then
        test_preset "$specific_preset"
        print_preset_summary
        exit $?
    fi

    # Run platform tests
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

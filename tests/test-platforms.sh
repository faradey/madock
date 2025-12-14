#!/usr/bin/env bash

# Test script for madock Magento platform
# Tests Magento presets to verify docker-compose generation and basic commands

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

# Test results
RESULTS_FILE="/tmp/madock-test-results.txt"
FAILED_TESTS=0
PASSED_TESTS=0
KEEP_PROJECT=false
INTERACTIVE_MODE=false

# Magento presets to test (from preset.go)
MAGENTO_PRESETS=(
    "Magento 2.4.8 (Latest)"
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

# Run command with TTY emulation (for docker exec -it)
run_with_tty() {
    local logfile="$1"
    shift
    case "$(uname)" in
        Linux)
            script -q -c "$*" "$logfile"
            ;;
        Darwin)
            script -q "$logfile" "$@"
            ;;
    esac
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

    # Remove project with containers, images and volumes
    if [[ -d "$project_dir" ]]; then
        cd "$project_dir"
        "$MADOCK_BIN" project:remove --force --name="$project_name" 2>/dev/null || true
    fi

    # Remove remaining directories (if project:remove didn't clean everything)
    rm -rf "$project_dir"
    rm -rf "$MADOCK_DIR/projects/$project_name"
    rm -rf "$MADOCK_DIR/aruntime/projects/$project_name"
    rm -rf "$MADOCK_DIR/cache/$project_name-proxy.conf"

    log_success "Cleaned up $project_name"
}

# Test a single preset
test_preset() {
    local preset_name="$1"
    local safe_name=$(echo "$preset_name" | tr ' +()./' '------' | tr -s '-' | tr '[:upper:]' '[:lower:]' | sed 's/-$//')
    local project_name="test-${safe_name}"
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

    # Create minimal composer.json
    cat > composer.json << 'EOF'
{
    "name": "test/magento2",
    "type": "project"
}
EOF

    # Run setup with preset
    # --yes flag handles confirmation prompts
    # run_with_tty provides TTY for docker exec during download/install
    log_info "Running madock setup with preset: $preset_name"
    if ! run_with_tty /tmp/madock-setup-preset.log "$MADOCK_BIN" setup -d -i -y --platform=magento2 --preset="$preset_name" --hosts="${host}:base"; then
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

    # Test service:enable command
    log_info "Testing service:enable phpmyadmin..."
    if "$MADOCK_BIN" service:enable phpmyadmin 2>&1; then
        log_success "Service:enable phpmyadmin works"
    else
        log_error "Service:enable phpmyadmin failed"
        test_passed=false
    fi

    # Cleanup (unless --keep flag is set)
    if ! $KEEP_PROJECT; then
        cleanup_project "$project_name"
    else
        log_info "Keeping project at: $project_dir"
        log_info "Config at: $MADOCK_DIR/projects/$project_name/config.xml"
    fi

    # Record result
    if $test_passed; then
        echo "$safe_name=PASSED" >> "$RESULTS_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "Preset '$preset_name': ALL TESTS PASSED"
    else
        echo "$safe_name=FAILED" >> "$RESULTS_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "Preset '$preset_name': SOME TESTS FAILED"
    fi

    echo ""
}

# Test all presets
test_all_presets() {
    log_header "Testing Magento Presets"

    for preset in "${MAGENTO_PRESETS[@]}"; do
        test_preset "$preset"
    done
}

# Interactive test - manual selection of options
test_interactive() {
    local project_name="test-interactive"
    local project_dir="$TMP_DIR/$project_name"
    local host="interactive.test"
    local test_passed=true

    log_header "Interactive Test (Manual Selection)"

    # Cleanup any previous test
    cleanup_project "$project_name"

    # Create project directory
    log_info "Creating project directory: $project_dir"
    mkdir -p "$project_dir"
    cd "$project_dir"

    # Create minimal composer.json
    cat > composer.json << 'EOF'
{
    "name": "test/magento2",
    "type": "project"
}
EOF

    # Run setup interactively (no preset, no auto-confirm)
    log_info "Running madock setup interactively..."
    log_info "Please answer the setup questions manually"
    echo ""
    if ! "$MADOCK_BIN" setup -d -i --platform=magento2 --hosts="${host}:base" 2>&1; then
        log_error "Setup failed"
        test_passed=false
    else
        log_success "Setup completed"
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

    # Test service:enable command
    log_info "Testing service:enable phpmyadmin..."
    if "$MADOCK_BIN" service:enable phpmyadmin 2>&1; then
        log_success "Service:enable phpmyadmin works"
    else
        log_error "Service:enable phpmyadmin failed"
        test_passed=false
    fi

    # Cleanup (unless --keep flag is set)
    if ! $KEEP_PROJECT; then
        cleanup_project "$project_name"
    else
        log_info "Keeping project at: $project_dir"
        log_info "Config at: $MADOCK_DIR/projects/$project_name/config.xml"
    fi

    # Record result
    if $test_passed; then
        echo "interactive=PASSED" >> "$RESULTS_FILE"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "Interactive test: ALL TESTS PASSED"
    else
        echo "interactive=FAILED" >> "$RESULTS_FILE"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "Interactive test: SOME TESTS FAILED"
    fi

    echo ""
}

# Print summary
print_summary() {
    log_header "TEST SUMMARY"

    local preset_count=${#MAGENTO_PRESETS[@]}
    echo -e "Total presets tested: $preset_count"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    echo ""

    echo "Results by preset:"
    for preset in "${MAGENTO_PRESETS[@]}"; do
        local safe_name=$(echo "$preset" | tr ' +()./' '------' | tr -s '-' | tr '[:upper:]' '[:lower:]' | sed 's/-$//')
        local result="NOT RUN"
        if [[ -f "$RESULTS_FILE" ]]; then
            result=$(grep "^$safe_name=" "$RESULTS_FILE" 2>/dev/null | cut -d= -f2 || echo "NOT RUN")
        fi
        if [[ "$result" == "PASSED" ]]; then
            echo -e "  ${GREEN}$preset: $result${NC}"
        else
            echo -e "  ${RED}$preset: $result${NC}"
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

    # Cleanup preset tests
    for preset in "${MAGENTO_PRESETS[@]}"; do
        local safe_name=$(echo "$preset" | tr ' +()./' '------' | tr -s '-' | tr '[:upper:]' '[:lower:]' | sed 's/-$//')
        cleanup_project "test-${safe_name}"
    done

    # Cleanup interactive test
    cleanup_project "test-interactive"

    # Remove tmp directory if empty
    if [[ -d "$TMP_DIR" ]] && [[ -z "$(ls -A "$TMP_DIR")" ]]; then
        rmdir "$TMP_DIR"
    fi

    rm -f "$RESULTS_FILE"

    log_success "Cleanup completed"
}

# Main
main() {
    local cleanup_only=false
    local specific_preset=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --preset)
                specific_preset="$2"
                shift 2
                ;;
            --keep)
                KEEP_PROJECT=true
                shift
                ;;
            --interactive|-i)
                INTERACTIVE_MODE=true
                shift
                ;;
            --cleanup)
                cleanup_only=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --preset NAME    Test specific preset (use quotes for names with spaces)"
                echo "  --interactive|-i Run interactive test with manual selection"
                echo "  --keep           Keep project after test (don't cleanup)"
                echo "  --cleanup        Only cleanup, don't run tests"
                echo "  --help           Show this help"
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

    log_header "MADOCK MAGENTO TESTS"
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

    # Run interactive test
    if $INTERACTIVE_MODE; then
        test_interactive
        print_summary
        exit $?
    fi

    # Run specific preset test
    if [[ -n "$specific_preset" ]]; then
        test_preset "$specific_preset"
    else
        # Run all preset tests
        test_all_presets
    fi

    print_summary
}

main "$@"

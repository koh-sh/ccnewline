#!/bin/bash

# ccnewline minimal functionality test script
# Tests Claude Code Edit/MultiEdit/Write patterns and output modes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CCNEWLINE="$PROJECT_ROOT/ccnewline"
TMP_DIR="$SCRIPT_DIR/tmp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to clean up test files
cleanup() {
    if [[ -d "$TMP_DIR" ]]; then
        chmod -R 755 "$TMP_DIR" 2>/dev/null
        rm -rf "$TMP_DIR"
    fi
}

# Function to create test file without newline
create_file_without_newline() {
    local filename="$1"
    local content="$2"
    printf "%s" "$content" > "$TMP_DIR/$filename"
}

# Function to create test file with newline
create_file_with_newline() {
    local filename="$1"
    local content="$2"
    printf "%s\n" "$content" > "$TMP_DIR/$filename"
}

# Function to check if file ends with newline
check_newline() {
    local filename="$1"
    local filepath="$TMP_DIR/$filename"
    if [[ ! -f "$filepath" ]]; then
        return 1
    fi
    
    # Use tail and od to check last byte
    if tail -c1 "$filepath" | od -An -tx1 | grep -q "0a"; then
        return 0  # Has newline
    else
        return 1  # No newline
    fi
}

# Simple test runner for Claude Code tool patterns and output modes
run_test() {
    local test_name="$1"
    local json_input="$2"
    local expected_file="$3"
    local mode="$4"  # "normal", "silent", "debug"
    
    print_status "Testing: $test_name"
    
    # Build command flags
    local cmd_flags=""
    case "$mode" in
        "silent") cmd_flags="-s" ;;
        "debug") cmd_flags="-d" ;;
        *) cmd_flags="" ;;
    esac
    
    # Execute and capture output
    local output_file="$TMP_DIR/test_output.txt"
    echo "$json_input" | "$CCNEWLINE" $cmd_flags > "$output_file" 2>&1
    local captured_output="$(cat "$output_file")"
    
    local test_passed=true
    
    # Check file was processed (if expected)
    if [[ -n "$expected_file" ]]; then
        if [[ ! -f "$TMP_DIR/$expected_file" ]]; then
            print_error "  ✗ Expected file $expected_file not found"
            test_passed=false
        elif ! check_newline "$expected_file"; then
            print_error "  ✗ File $expected_file should end with newline"
            test_passed=false
        fi
    fi
    
    # Check output based on mode
    case "$mode" in
        "silent")
            if [[ -n "$captured_output" ]]; then
                print_error "  ✗ Silent mode produced output: '$captured_output'"
                test_passed=false
            fi
            ;;
        "debug")
            if [[ ! "$captured_output" =~ "INPUT PARSING" ]] || [[ ! "$captured_output" =~ "PROCESSING" ]]; then
                print_error "  ✗ Debug output missing expected sections"
                test_passed=false
            fi
            ;;
        "normal")
            if [[ -n "$expected_file" ]]; then
                # Normal mode should output "Added newline to [file]" when newline is added
                if [[ ! "$captured_output" =~ "Added newline to" ]]; then
                    print_error "  ✗ Normal mode should output 'Added newline to' message"
                    test_passed=false
                fi
            else
                # No expected file means file already had newline, should produce no output
                if [[ "$captured_output" =~ "Added newline to" ]]; then
                    print_error "  ✗ Normal mode should not output when file already has newline"
                    test_passed=false
                fi
            fi
            ;;
    esac
    
    if [[ "$test_passed" == "true" ]]; then
        print_success "  ✓ PASS"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        print_error "  ✗ FAIL"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    
    TESTS_RUN=$((TESTS_RUN + 1))
    rm -f "$output_file"
}


# Test execution tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Main test function
main() {
    print_status "Starting ccnewline minimal functionality tests"
    
    # Check if ccnewline binary exists
    if [[ ! -f "$CCNEWLINE" ]]; then
        print_status "Building ccnewline..."
        cd "$PROJECT_ROOT" && go build -o ccnewline
        if [[ ! -f "$CCNEWLINE" ]]; then
            print_error "Failed to build ccnewline"
            exit 1
        fi
    fi
    
    # Setup test environment
    cleanup
    mkdir -p "$TMP_DIR"
    cd "$TMP_DIR"
    
    # ============================================================================
    # CLAUDE CODE TOOL PATTERN TESTS (Edit, MultiEdit, Write)
    # ============================================================================
    print_status "=== CLAUDE CODE TOOL PATTERN TESTS ==="
    
    # Test Edit pattern (file_path field)
    create_file_without_newline "edit_test.txt" "edit content"
    run_test "Edit pattern" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/edit_test.txt\"}}" \
        "edit_test.txt" \
        "normal"
    
    # Test MultiEdit pattern (paths array)
    create_file_without_newline "multi1.txt" "multi content 1"
    create_file_without_newline "multi2.txt" "multi content 2"
    run_test "MultiEdit pattern" \
        "{\"tool_input\": {\"paths\": [\"$TMP_DIR/multi1.txt\", \"$TMP_DIR/multi2.txt\"]}}" \
        "multi1.txt" \
        "normal"
    
    # Test Write pattern (file_path field, same as Edit)
    create_file_without_newline "write_test.txt" "write content"
    run_test "Write pattern" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/write_test.txt\"}}" \
        "write_test.txt" \
        "normal"
    
    # ============================================================================
    # OUTPUT MODE TESTS (normal, silent, debug)
    # ============================================================================
    print_status "=== OUTPUT MODE TESTS ==="
    
    # Test normal mode - no newline (should add)
    create_file_without_newline "normal_mode.txt" "normal content"
    run_test "Normal mode - no newline" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/normal_mode.txt\"}}" \
        "normal_mode.txt" \
        "normal"
    
    # Test normal mode - has newline (should not add)
    create_file_with_newline "normal_has_newline.txt" "normal content"
    run_test "Normal mode - has newline" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/normal_has_newline.txt\"}}" \
        "" \
        "normal"
    
    # Test silent mode
    create_file_without_newline "silent_mode.txt" "silent content"
    run_test "Silent mode" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/silent_mode.txt\"}}" \
        "silent_mode.txt" \
        "silent"
    
    # Test debug mode
    create_file_without_newline "debug_mode.txt" "debug content"
    run_test "Debug mode" \
        "{\"tool_input\": {\"file_path\": \"$TMP_DIR/debug_mode.txt\"}}" \
        "debug_mode.txt" \
        "debug"
    
    # ============================================================================
    # SUMMARY
    # ============================================================================
    
    cleanup
    
    echo
    print_status "=== TEST SUMMARY ==="
    print_status "Tests run: $TESTS_RUN"
    if [[ "$TESTS_FAILED" -eq 0 ]]; then
        print_success "All $TESTS_PASSED tests PASSED ✓"
        return 0
    else
        print_error "$TESTS_FAILED tests FAILED ✗"
        return 1
    fi
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"
exit_code=$?
exit $exit_code

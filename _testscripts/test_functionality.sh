#!/bin/bash

# Simple integration tests for ccnewline pattern matching feature
# Tests basic functionality and new --exclude/--include options

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CCNEWLINE="$PROJECT_ROOT/ccnewline"
TMP_DIR="$SCRIPT_DIR/tmp"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Test counter
TESTS=0
PASSED=0

# Helper functions
cleanup() {
    rm -rf "$TMP_DIR"
}

pass() {
    echo -e "${GREEN}✓${NC} $1"
    PASSED=$((PASSED + 1))
}

fail() {
    echo -e "${RED}✗${NC} $1"
}

run_test() {
    TESTS=$((TESTS + 1))
    echo "Test $TESTS: $1"
}

# Setup
trap cleanup EXIT
cleanup
mkdir -p "$TMP_DIR"

# Build if needed
if [[ ! -f "$CCNEWLINE" ]]; then
    cd "$PROJECT_ROOT" && go build -o ccnewline
fi

echo "Running ccnewline integration tests..."
echo

# Test 1: Basic functionality - file without newline
run_test "Basic functionality - adds newline to file without one"
printf "test content" > "$TMP_DIR/test1.txt"
echo '{"tool_input": {"file_path": "'$TMP_DIR'/test1.txt"}}' | "$CCNEWLINE" > /dev/null
if tail -c1 "$TMP_DIR/test1.txt" | od -An -tx1 | grep -q "0a"; then
    pass "File now ends with newline"
else
    fail "File should end with newline"
fi
echo

# Test 2: File already has newline - should not modify
run_test "File with newline - should not be modified"
printf "test content\n" > "$TMP_DIR/test2.txt"
before=$(stat -f%m "$TMP_DIR/test2.txt" 2>/dev/null || stat -c%Y "$TMP_DIR/test2.txt")
echo '{"tool_input": {"file_path": "'$TMP_DIR'/test2.txt"}}' | "$CCNEWLINE" > /dev/null
after=$(stat -f%m "$TMP_DIR/test2.txt" 2>/dev/null || stat -c%Y "$TMP_DIR/test2.txt")
if [[ "$before" == "$after" ]]; then
    pass "File was not modified"
else
    fail "File should not have been modified"
fi
echo

# Test 3: Exclude pattern - should exclude .txt files
run_test "Exclude pattern - excludes .txt files"
printf "go content" > "$TMP_DIR/test3.go"
printf "txt content" > "$TMP_DIR/test3.txt"
echo '{"tool_input": {"paths": ["'$TMP_DIR'/test3.go", "'$TMP_DIR'/test3.txt"]}}' | "$CCNEWLINE" --exclude "*.txt" > /dev/null

# Check both conditions for this test
go_processed=false
txt_excluded=false

if tail -c1 "$TMP_DIR/test3.go" | od -An -tx1 | grep -q "0a"; then
    go_processed=true
fi

if ! tail -c1 "$TMP_DIR/test3.txt" | od -An -tx1 | grep -q "0a"; then
    txt_excluded=true
fi

if [[ "$go_processed" == "true" && "$txt_excluded" == "true" ]]; then
    pass "Exclude pattern works correctly"
else
    fail "Exclude pattern failed"
fi
echo

# Test 4: Include pattern - should only include .go files
run_test "Include pattern - only processes .go files"
printf "go content" > "$TMP_DIR/test4.go"
printf "txt content" > "$TMP_DIR/test4.txt"
echo '{"tool_input": {"paths": ["'$TMP_DIR'/test4.go", "'$TMP_DIR'/test4.txt"]}}' | "$CCNEWLINE" --include "*.go" > /dev/null

# Check both conditions for this test
go_processed=false
txt_excluded=false

if tail -c1 "$TMP_DIR/test4.go" | od -An -tx1 | grep -q "0a"; then
    go_processed=true
fi

if ! tail -c1 "$TMP_DIR/test4.txt" | od -An -tx1 | grep -q "0a"; then
    txt_excluded=true
fi

if [[ "$go_processed" == "true" && "$txt_excluded" == "true" ]]; then
    pass "Include pattern works correctly"
else
    fail "Include pattern failed"
fi
echo

# Test 5: Multiple exclude patterns
run_test "Multiple exclude patterns - excludes .txt and .md files"
printf "go content" > "$TMP_DIR/test5.go"
printf "txt content" > "$TMP_DIR/test5.txt"
printf "md content" > "$TMP_DIR/test5.md"
echo '{"tool_input": {"paths": ["'$TMP_DIR'/test5.go", "'$TMP_DIR'/test5.txt", "'$TMP_DIR'/test5.md"]}}' | "$CCNEWLINE" --exclude "*.txt,*.md" > /dev/null

# Check all conditions for this test
go_processed=false
files_excluded=true

if tail -c1 "$TMP_DIR/test5.go" | od -An -tx1 | grep -q "0a"; then
    go_processed=true
fi

if tail -c1 "$TMP_DIR/test5.txt" | od -An -tx1 | grep -q "0a"; then
    files_excluded=false
fi
if tail -c1 "$TMP_DIR/test5.md" | od -An -tx1 | grep -q "0a"; then
    files_excluded=false
fi

if [[ "$go_processed" == "true" && "$files_excluded" == "true" ]]; then
    pass "Multiple exclude patterns work correctly"
else
    fail "Multiple exclude patterns failed"
fi
echo

# Test 6: Mutual exclusivity check - should fail with both flags
run_test "Mutual exclusivity - should fail with both --exclude and --include"
if echo '{"tool_input": {"file_path": "'$TMP_DIR'/test.txt"}}' | "$CCNEWLINE" --exclude "*.txt" --include "*.go" 2>/dev/null; then
    fail "Should have failed with both flags"
else
    pass "Correctly failed with both --exclude and --include flags"
fi
echo

# Summary
echo "================================"
echo "Tests completed: $TESTS"
echo "Tests passed: $PASSED"
echo "Tests failed: $((TESTS - PASSED))"

if [[ "$PASSED" == "$TESTS" ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
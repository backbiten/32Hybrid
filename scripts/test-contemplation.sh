#!/usr/bin/env bash
# test-contemplation.sh - Test script for the contemplation period implementation

set -e

echo "======================================"
echo "32Hybrid Contemplation Period Test"
echo "======================================"
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to print test results
print_test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $2"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Helper function to cleanup
cleanup() {
    echo -e "\n${YELLOW}Cleaning up test files...${NC}"
    rm -f /tmp/contemplation_progress /tmp/neural_registry_unlocked
    killall -9 contemplation 2>/dev/null || true
    killall -9 contemplation-demo 2>/dev/null || true
}

# Set up cleanup on exit
trap cleanup EXIT

echo "Step 1: Building C implementation..."
cd legacy32
make clean > /dev/null 2>&1 || true
if make > /dev/null 2>&1; then
    print_test_result 0 "C implementation builds successfully"
else
    print_test_result 1 "C implementation build failed"
    exit 1
fi
cd ..

echo
echo "Step 2: Testing Go packages..."

# Test winstratch package
if go test -v ./internal/winstratch/ 2>&1 | grep -q "no test files"; then
    echo -e "${YELLOW}⊘${NC} WinStratch package has no tests (expected)"
else
    if go test ./internal/winstratch/ > /dev/null 2>&1; then
        print_test_result 0 "WinStratch package tests pass"
    else
        print_test_result 1 "WinStratch package tests failed"
    fi
fi

# Test teacher package
if go test -v ./internal/teacher/ 2>&1 | grep -q "no test files"; then
    echo -e "${YELLOW}⊘${NC} Teacher package has no tests (expected)"
else
    if go test ./internal/teacher/ > /dev/null 2>&1; then
        print_test_result 0 "Teacher package tests pass"
    else
        print_test_result 1 "Teacher package tests failed"
    fi
fi

echo
echo "Step 3: Building Go demo application..."
if go build -o /tmp/contemplation-demo ./cmd/contemplation-demo/ > /dev/null 2>&1; then
    print_test_result 0 "Go demo application builds successfully"
else
    print_test_result 1 "Go demo application build failed"
    exit 1
fi

echo
echo "Step 4: Testing file creation..."
cleanup

# Start C contemplation in background (with short timeout)
timeout 5 ./legacy32/contemplation > /dev/null 2>&1 &
CPID=$!
sleep 2

if [ -f /tmp/contemplation_progress ]; then
    print_test_result 0 "Progress file is created"
else
    print_test_result 1 "Progress file not created"
fi

# Check progress file format
if [ -f /tmp/contemplation_progress ]; then
    LINES=$(wc -l < /tmp/contemplation_progress)
    if [ "$LINES" -eq 3 ]; then
        print_test_result 0 "Progress file has correct format (3 lines)"
    else
        print_test_result 1 "Progress file format incorrect (expected 3 lines, got $LINES)"
    fi
fi

# Kill the background process
kill $CPID 2>/dev/null || true
cleanup

echo
echo "Step 5: Testing quick completion..."
# Run with skip-wait flag
if timeout 5 /tmp/contemplation-demo --skip-wait > /dev/null 2>&1; then
    print_test_result 0 "Demo runs successfully with skip-wait"
else
    print_test_result 1 "Demo failed with skip-wait"
fi

echo
echo "Step 6: Simulating short contemplation period..."
# Modify the C code temporarily to use a shorter duration (not done in this test)
# Instead, we'll just verify the structure is correct

if [ -f legacy32/contemplation.c ]; then
    if grep -q "CONTEMPLATION_DURATION_SEC" legacy32/contemplation.c; then
        print_test_result 0 "C code has configurable duration constant"
    else
        print_test_result 1 "C code missing duration constant"
    fi
fi

echo
echo "Step 7: Checking documentation..."
if [ -f docs/contemplation-period.md ]; then
    print_test_result 0 "Documentation exists"
else
    print_test_result 1 "Documentation missing"
fi

echo
echo "Step 8: Verifying API completeness..."

# Check for required C functions
for func in "hold_ai_until_ready" "is_neural_registry_locked" "release_neural_registry_lock" "verify_i386_knowledge"; do
    if grep -q "$func" legacy32/contemplation.h; then
        print_test_result 0 "C API has $func()"
    else
        print_test_result 1 "C API missing $func()"
    fi
done

# Check for required Go functions
if grep -q "ShowContemplationDialog" internal/winstratch/contemplation.go; then
    print_test_result 0 "Go WinStratch API has ShowContemplationDialog()"
else
    print_test_result 1 "Go WinStratch API missing ShowContemplationDialog()"
fi

if grep -q "CanAccessMicroBus" internal/teacher/teacher.go; then
    print_test_result 0 "Go Teacher API has CanAccessMicroBus()"
else
    print_test_result 1 "Go Teacher API missing CanAccessMicroBus()"
fi

echo
echo "======================================"
echo "Test Summary"
echo "======================================"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi

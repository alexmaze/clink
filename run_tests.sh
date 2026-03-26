#!/usr/bin/env bash
#
# run_tests.sh - Build container test image and run all tests with report output
# Supports both podman and docker (prefers podman)
#
set -euo pipefail

# Auto-detect container runtime
if command -v podman &>/dev/null; then
    RUNTIME="podman"
elif command -v docker &>/dev/null; then
    RUNTIME="docker"
else
    echo "Error: neither podman nor docker found" >&2
    exit 1
fi

IMAGE_NAME="clink-test"
CONTAINER_NAME="clink-test-runner"
REPORT_FILE="test_report.txt"

cd "$(dirname "$0")"

echo "=========================================="
echo "  clink Container Test Runner ($RUNTIME)"
echo "=========================================="
echo ""

# 1. Build the test image
echo "▶ Building test image..."
$RUNTIME build -f Dockerfile.test -t "$IMAGE_NAME" .
echo "✔ Image built successfully"
echo ""

# 2. Remove old container if exists
$RUNTIME rm -f "$CONTAINER_NAME" 2>/dev/null || true

# 3. Run tests
echo "▶ Running tests in container..."
echo ""

# Run tests and capture output (allow non-zero exit for reporting)
set +e
$RUNTIME run \
    --name "$CONTAINER_NAME" \
    "$IMAGE_NAME" \
    sh -c '
        echo "=== Go Version ==="
        go version
        echo ""

        echo "=== Running Tests ==="
        go test -v -count=1 -coverprofile=/tmp/coverage.out ./... 2>&1
        TEST_EXIT=$?
        echo ""

        echo "=== Coverage Summary ==="
        if [ -f /tmp/coverage.out ]; then
            go tool cover -func=/tmp/coverage.out 2>/dev/null || echo "(coverage data unavailable)"
        fi
        echo ""

        exit $TEST_EXIT
    ' | tee "$REPORT_FILE"

TEST_EXIT=${PIPESTATUS[0]}
set -e

echo ""
echo "=========================================="
if [ $TEST_EXIT -eq 0 ]; then
    echo "  ✔ ALL TESTS PASSED"
else
    echo "  ✘ SOME TESTS FAILED (exit code: $TEST_EXIT)"
fi
echo "=========================================="
echo ""
echo "▶ Full report saved to: $REPORT_FILE"

# 4. Cleanup
$RUNTIME rm -f "$CONTAINER_NAME" 2>/dev/null || true

exit $TEST_EXIT

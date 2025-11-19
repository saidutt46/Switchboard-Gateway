#!/bin/bash
# Run All Rate Limiting Load Tests

set -e

RESULTS_DIR="tests/results"
mkdir -p "$RESULTS_DIR"

echo "🚀 Rate Limiting Load Tests"
echo "==========================="
echo ""

# Test 1: Token Bucket
echo "📊 Test 1: Token Bucket Algorithm"
echo "----------------------------------"
./tests/load/switch_algorithm.sh token-bucket
sleep 3

echo "Running Token Bucket load test..."
k6 run tests/load/rate_limit_token_bucket.js \
  --out json="$RESULTS_DIR/token_bucket_raw.json" \
  --summary-export="$RESULTS_DIR/token_bucket_summary.json"

echo ""
echo "✅ Token Bucket test complete!"
echo ""
echo "Waiting 70 seconds before next test..."
sleep 70

# Test 2: Sliding Window
echo "📊 Test 2: Sliding Window Algorithm"
echo "------------------------------------"
./tests/load/switch_algorithm.sh sliding-window
sleep 3

echo "Running Sliding Window load test..."
k6 run tests/load/rate_limit_sliding_window.js \
  --out json="$RESULTS_DIR/sliding_window_raw.json" \
  --summary-export="$RESULTS_DIR/sliding_window_summary.json"

echo ""
echo "✅ Sliding Window test complete!"
echo ""

# Generate comparison report
echo "📈 Generating Comparison Report"
echo "--------------------------------"
cat > "$RESULTS_DIR/comparison_report.md" << 'EOF'
# Rate Limiting Load Test Results

## Test Summary

### Token Bucket Algorithm
- **Purpose**: Test gradual refill and burst tolerance
- **Configuration**: 10 requests/minute, 0.167 tokens/second refill
- **Results**: See `token_bucket_summary.json`

### Sliding Window Algorithm
- **Purpose**: Test strict enforcement and exact limits
- **Configuration**: 10 requests/minute, no gradual refill
- **Results**: See `sliding_window_summary.json`

## Key Metrics

| Metric | Token Bucket | Sliding Window |
|--------|--------------|----------------|
| Burst Allowed | 10-11 | Exactly 10 |
| Gradual Refill | Yes | No |
| Strict Enforcement | Moderate | High |
| Burst Tolerance | High | None |

## Files Generated
- `token_bucket_raw.json` - Detailed Token Bucket results
- `token_bucket_summary.json` - Token Bucket summary
- `sliding_window_raw.json` - Detailed Sliding Window results
- `sliding_window_summary.json` - Sliding Window summary

## Viewing Results
```bash
# View Token Bucket summary
cat tests/results/token_bucket_summary.json | jq

# View Sliding Window summary
cat tests/results/sliding_window_summary.json | jq

# Compare metrics
jq '.metrics' tests/results/token_bucket_summary.json
jq '.metrics' tests/results/sliding_window_summary.json
```
EOF

echo "✅ Comparison report generated: $RESULTS_DIR/comparison_report.md"
echo ""
echo "🎉 All tests complete!"
echo ""
echo "📁 Results saved in: $RESULTS_DIR/"
echo ""
echo "View results:"
echo "  cat $RESULTS_DIR/comparison_report.md"
echo "  cat $RESULTS_DIR/token_bucket_summary.json | jq"
echo "  cat $RESULTS_DIR/sliding_window_summary.json | jq"
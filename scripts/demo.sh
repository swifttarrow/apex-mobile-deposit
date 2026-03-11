#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "========================================"
echo "Checkstream Mobile Check Deposit Demo"
echo "========================================"
echo ""

# Helper function
request() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local extra_headers="${4:-}"

  if [ -n "$body" ]; then
    curl -s -X "$method" "$BASE_URL$path" \
      -H "Content-Type: application/json" \
      $extra_headers \
      -d "$body" | jq .
  else
    curl -s -X "$method" "$BASE_URL$path" \
      $extra_headers | jq .
  fi
}

echo "--- Health Check ---"
request GET /health
echo ""

echo "--- Scenario 1: Clean Pass (ACC-001, \$150) ---"
CLEAN=$(curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACC-001","amount":150.00,"front_image":"base64_front","back_image":"base64_back"}')
echo "$CLEAN" | jq .
CLEAN_ID=$(echo "$CLEAN" | jq -r '.id')
echo ""

echo "--- Get Deposit Status ---"
request GET "/deposits/$CLEAN_ID"
echo ""

echo "--- Scenario 2: IQA Blur Fail (ACC-IQA-BLUR) ---"
request POST /deposits '{"account_id":"ACC-IQA-BLUR","amount":100.00,"front_image":"blurry","back_image":"blurry"}'
echo ""

echo "--- Scenario 3: IQA Glare Fail (ACC-IQA-GLARE) ---"
request POST /deposits '{"account_id":"ACC-IQA-GLARE","amount":100.00,"front_image":"glare","back_image":"glare"}'
echo ""

echo "--- Scenario 4: MICR Fail → Flagged for Review ---"
MICR=$(curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACC-MICR-FAIL","amount":200.00,"front_image":"front","back_image":"back"}')
echo "$MICR" | jq .
MICR_ID=$(echo "$MICR" | jq -r '.transfer.id')
echo ""

echo "--- Operator Queue ---"
request GET /operator/queue
echo ""

echo "--- Operator Approves MICR Fail ---"
request POST /operator/approve "{\"transfer_id\":\"$MICR_ID\",\"operator_id\":\"op-001\",\"note\":\"manually verified MICR\"}"
echo ""

echo "--- Scenario 5: Amount Mismatch → Flagged ---"
MISMATCH=$(curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACC-MISMATCH","amount":1500.00,"front_image":"front","back_image":"back"}')
echo "$MISMATCH" | jq .
MISMATCH_ID=$(echo "$MISMATCH" | jq -r '.transfer.id')
echo ""

echo "--- Operator Rejects Amount Mismatch ---"
request POST /operator/reject "{\"transfer_id\":\"$MISMATCH_ID\",\"operator_id\":\"op-001\",\"note\":\"amount mismatch confirmed\"}"
echo ""

echo "--- Scenario 6: Over Deposit Limit (ACC-OVER-LIMIT, \$6000) ---"
request POST /deposits '{"account_id":"ACC-OVER-LIMIT","amount":6000.00,"front_image":"front","back_image":"back"}'
echo ""

echo "--- Scenario 7: Duplicate Detected (ACC-DUP-001) ---"
request POST /deposits '{"account_id":"ACC-DUP-001","amount":100.00,"front_image":"front","back_image":"back"}'
echo ""

echo "--- Scenario 8: Retirement Account (ACC-RETIRE-001) ---"
request POST /deposits '{"account_id":"ACC-RETIRE-001","amount":1000.00,"front_image":"front","back_image":"back"}'
echo ""

echo "--- Scenario 9: Return/Reversal ---"
RETURN_DEPOSIT=$(curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -d '{"account_id":"ACC-001","amount":500.00,"front_image":"front","back_image":"back"}')
RETURN_ID=$(echo "$RETURN_DEPOSIT" | jq -r '.id')
echo "Deposited transfer: $RETURN_ID"
echo ""

request POST /returns "{\"transfer_id\":\"$RETURN_ID\",\"reason\":\"insufficient funds\"}"
echo ""

echo "--- Scenario 10: Idempotency ---"
IDEM_KEY="demo-idem-$(date +%s)"
echo "Submitting deposit with idempotency key: $IDEM_KEY"
curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $IDEM_KEY" \
  -d '{"account_id":"ACC-001","amount":75.00,"front_image":"front","back_image":"back"}' | jq .
echo ""
echo "Replaying same idempotency key..."
curl -s -X POST "$BASE_URL/deposits" \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: $IDEM_KEY" \
  -d '{"account_id":"ACC-001","amount":75.00,"front_image":"front","back_image":"back"}' | jq .
echo ""

echo "--- Scenario 11: Settlement Trigger ---"
request POST /settlement/trigger
echo ""

echo "--- Ledger Entries ---"
request GET /ledger
echo ""

echo "--- Account Balance (ACC-001) ---"
request GET /accounts/ACC-001/balance
echo ""

echo "========================================"
echo "Demo Complete!"
echo "========================================"

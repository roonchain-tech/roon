#!/usr/bin/env bash
set -euo pipefail

# Usage: ./error_edge_case.sh
# Requires in .env: CUSTOM_RPC, PRIVATE_KEY, ACCOUNT_2 (recipient), CONTRACT
# shellcheck source=../.env
# shellcheck disable=SC1091
source ../.env
export FOUNDRY_DISABLE_NIGHTLY_WARNING=1

RPC_URL=${CUSTOM_RPC:-http://127.0.0.1:8545}
PK=${PRIVATE_KEY:?}
RECIPIENT=${ACCOUNT_2:?}
CHAIN_ID=${CHAIN_ID:-1492}

# Ensure CONTRACT is set
if [ -z "${CONTRACT:-}" ]; then
	echo "Error: CONTRACT environment variable not set."
	exit 1
fi

# Amount exceeding typical balance (2000 tokens)
AMOUNT=2000000000000000000000

echo "🔄 Sending transfer exceeding balance..."

# 1) Send via cast and capture output (suppress exit)
echo "❌ Attempting transfer that should fail:"
echo "$ cast send \"$CONTRACT\" 'transfer(address,uint256)' \"$RECIPIENT\" \"$AMOUNT\" --rpc-url \"$RPC_URL\" --private-key \"[HIDDEN]\" --chain-id \"$CHAIN_ID\" --json"
OUTPUT=$(cast send \
	"$CONTRACT" \
	'transfer(address,uint256)' "$RECIPIENT" "$AMOUNT" \
	--rpc-url "$RPC_URL" \
	--private-key "$PK" \
	--chain-id "$CHAIN_ID" \
	--json 2>&1 || true)

# 2) A revert is expected. Check the raw output for revert markers so both
#    cast's plain-text format and its JSON error format are handled
#    (JSON errors wrap the message inside a "message" field).
if echo "$OUTPUT" | grep -q -e 'execution reverted' -e 'ERC20InsufficientBalance'; then
	echo "✅ Transaction reverted as expected"
	echo
	echo "Revert detail:"
	# Plain-text format: show the "Error:" line
	DETAIL=$(echo "$OUTPUT" | sed -n 's/.*\(Error:.*\)/\1/p')
	# JSON format: show the revert reason from the message field
	if [ -z "$DETAIL" ]; then
		DETAIL=$(echo "$OUTPUT" | grep -o 'ERC20InsufficientBalance([^)]*)' | head -1)
	fi
	if [ -z "$DETAIL" ]; then
		DETAIL=$(echo "$OUTPUT" | grep -o '"message":"[^"]*"' | head -1 | sed 's/^"message":"//;s/"$//')
	fi
	echo "${DETAIL:-execution reverted}"
else
	echo "❌ Expected a revert, but transaction succeeded or got an unexpected response:"
	echo
	echo "$OUTPUT"
	exit 1
fi

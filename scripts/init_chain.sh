#!/usr/bin/env bash
set -euo pipefail

HOME_DIR=${HOME_DIR:-$HOME/.evmd}
FUND_HEX=${FUND_HEX:-}
FUND_AMOUNT_ROON=${FUND_AMOUNT_ROON:-}
NO_DEV_FUNDS=${NO_DEV_FUNDS:-false}
CHAIN_ID=${CHAIN_ID:-9001}

export PATH="$(go env GOPATH 2>/dev/null)/bin:${PATH}"

command -v evmd >/dev/null 2>&1 || { echo "evmd not found"; exit 1; }

ARGS=("-y")
if [ "$NO_DEV_FUNDS" = "true" ]; then
  ARGS+=("--no-dev-funds")
fi
if [ -n "$FUND_HEX" ] && [ -n "$FUND_AMOUNT_ROON" ]; then
  ARGS+=("--fund-hex" "$FUND_HEX" "--fund-amount-roon" "$FUND_AMOUNT_ROON")
fi

CHAIN_ID="$CHAIN_ID" "$(dirname "$0")/../local_node.sh" "${ARGS[@]}"

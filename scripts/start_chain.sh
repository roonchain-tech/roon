#!/usr/bin/env bash
set -euo pipefail

HOME_DIR=${HOME_DIR:-$HOME/.evmd}
CHAIN_ID=${CHAIN_ID:-9001}
EVM_CHAIN_ID=${EVM_CHAIN_ID:-1492}
JSON_RPC_ADDR=${JSON_RPC_ADDR:-0.0.0.0:8545}
JSON_WS_ADDR=${JSON_WS_ADDR:-0.0.0.0:8546}

export PATH="$(go env GOPATH 2>/dev/null)/bin:${PATH}"

command -v evmd >/dev/null 2>&1 || { echo "evmd not found"; exit 1; }

exec evmd start \
  --home "$HOME_DIR" \
  --api.enabled-unsafe-cors true \
  --json-rpc.enable true \
  --json-rpc.address "$JSON_RPC_ADDR" \
  --json-rpc.ws-address "$JSON_WS_ADDR" \
  --minimum-gas-prices=50000000000aroon \
  --evm.mempool.price-limit=50000000000 \
  --evm.min-tip=0 \
  --chain-id "$CHAIN_ID" \
  --evm.evm-chain-id "$EVM_CHAIN_ID"

#!/usr/bin/env bash
set -euo pipefail
pkill -f 'evmd start' || true
pkill -f local_node.sh || true

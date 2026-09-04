#!/usr/bin/make -f

# Poll-based upper bound for node start / chain events; generous margin for
# slow GitHub-hosted runners (the framework returns as soon as the node is up).
WAIT_TIME ?= 60s

test:
	go test -timeout=30m -mod=readonly -tags='system_test' -v ./... --wait-time=$(WAIT_TIME) --binary evmd --chain-id local-4221 --bech32=roon

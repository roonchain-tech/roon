# Roon Chain

Roon is a sovereign Proof-of-Stake blockchain with built-in EVM compatibility.
It combines a native layer (CometBFT consensus, IBC, staking) with a fully
featured Ethereum VM layer: Solidity smart contracts, Ethereum JSON-RPC, and
native support for EVM wallets such as MetaMask and Rabby.

## Chain Information

| Option        | Value                          |
|---------------|--------------------------------|
| Binary        | `evmd`                         |
| Chain ID      | `9001` (native) / `1492` (EVM) |
| Denomination  | `aroon` (display: ROON)        |
| Bech32 prefix | `roon` / `roonvaloper`         |
| Consensus     | CometBFT (PoS)                 |

## Run a Local Node

Requirements: [Go](https://go.dev/) 1.25+, `jq`.

```bash
make install        # builds and installs the evmd binary
./local_node.sh     # initializes and starts a single-node local testnet
```

The script generates a fresh validator key and saves the mnemonic to
`$HOME/.evmd/val_mnemonic.txt`. To recover a specific validator key, provide
it via the environment variable:

```bash
VAL_MNEMONIC="your mnemonic words ..." ./local_node.sh
```

Optional flags:

- `-y`: overwrite previous chain data without prompting
- `-n`: do not overwrite previous chain data
- `--no-install`: skip `make install`
- `--additional-users N`: create N extra funded dev accounts
- `--mnemonics-input PATH`: provide custom dev account mnemonics (YAML list)
- `--fund-hex ADDRESS --fund-amount-roon N`: fund a hex address at genesis

Ethereum JSON-RPC is exposed at `http://localhost:8545` and WebSocket at
`ws://localhost:8546`.

## Mainnet

| Option                | Value                              |
|-----------------------|------------------------------------|
| RPC                   | `https://rpc.roonchain.com`        |
| Chain ID              | `1492`                             |
| Currency symbol       | `ROON`                             |
| Block explorer        | <https://browser.roonchain.com/>   |

### Connect a Wallet

Using MetaMask as an example:

1. Add a custom network with RPC URL `https://rpc.roonchain.com`
2. Chain ID: `1492`
3. Currency symbol: `ROON`
4. (Optional) Block explorer URL: `https://browser.roonchain.com`

For a local development node, use RPC `http://localhost:8545` instead;
for local development accounts, see `roon/README.md`.

## Testing

All test commands are defined in the `Makefile`:

```bash
make test-unit        # unit tests
make test-unit-cover  # coverage report
make test-fuzz        # fuzz tests
make test-solidity    # Solidity contract tests
make benchmark        # benchmarks
```

## Project Structure

- `roon/` – the chain application (wiring, genesis, upgrades, CLI entry point)
- `x/vm/` – the EVM module
- `precompiles/` – EVM precompiles exposing native chain functionality to Solidity
- `rpc/` – Ethereum JSON-RPC server
- `ante/` – transaction decorator logic
- `contracts/` – Solidity contracts used by the chain and tests

## License & Credits

Roon is fully open-source under the Apache 2.0 license (see
[LICENSE](./LICENSE) and [NOTICE](./NOTICE)).

## Contributing

Bug reports and feature requests are welcome via the
[issue tracker](https://github.com/roonchain-tech/roon/issues). For code
contributions, please read the [contributing guide](./CONTRIBUTING.md).

# prover-network-ops

Docs and utilities for operating within the Brevis Network prover marketplace, with an emphasis on the step-by-step manual for standing up a prover node. You will find the node operations guide plus helper tooling in this repo.

## Repository Layout

- `docs/` – additional walkthroughs (e.g., running bidder nodes, requesting proofs).
- `node-configs/` – sample configuration for network nodes.
- `tools/` – Go CLI that bundles the operational commands.

## Operation Docs

Review these guides first—they walk through the end-to-end workflows before you touch the CLI.

1. [`docs/run_prover_node.md`](docs/run_prover_node.md) – manual for provisioning a prover node: pico proving service setup (GPU/CPU) plus bidder service deployment.
2. [`docs/tools.md`](docs/tools.md) – operational cookbook for every CLI command (init prover, stake/unstake, claim commissions, request proofs, refund requests).
3. [`docs/request_proof.md`](docs/request_proof.md) – explains how to build Pico apps, generate VK/input/public values, and ultimately submit proof requests.

Once those make sense, use the sections below for reference details (configs and command synopsis).

## CLI Reference

See [`docs/tools.md`](docs/tools.md) for the complete operational guide (prerequisites, config fields, and usage examples). This section only lists the essentials:

- Build once inside `tools/`: `go build -o bin/tools ./...`
- Core commands (all flag/config details live in the doc above):

| Command | Description |
| ------- | ----------- |
| `stake` | Initialize prover/submitter accounts and stake tokens via `StakingController`. |
| `claim-commission` | Prover claims accumulated commission from `StakingController`. |
| `init-prover` | Initialize a prover profile (optionally linking a submitter) without staking. |
| `unstake` | Request or complete an unstake from a prover. |
| `request-proof` | Submit proof requests to `BrevisMarket`. |
| `refund` | Refund one or all unfulfilled requests via `BrevisMarket`. |

`tools/config.toml` provides a shared template for the commands that require signer info, staking params, or refund inputs. Update the sample sections (`[refund]`, `[init_prover]`, `[stake]`, `[unstake]`, etc.) before invoking the CLI. The former viewer tooling now lives in a separate repository.

## License

Copyright © 2025 Brevis Network. All rights reserved.

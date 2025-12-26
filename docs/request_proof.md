# How to Request a Proof

This guide covers proof submissions for the mainnet beta launch.

## Write the Application and Prepare Inputs

### Install Pico CLI

Use Pico CLI `v1.2.2` for the steps below.

#### Option 1: Install via Cargo

```
cargo +nightly-2025-08-04 install --git https://github.com/brevis-network/pico --tag v1.2.2 pico-cli
```

Verify the installation:

```
cargo pico --version
```

#### Option 2: Install from a Local Checkout

```
git clone https://github.com/brevis-network/pico
cd pico
git checkout v1.2.2
cd sdk/cli
cargo install --locked --force --path .
```

### Build the Application (ELF Artifact)

Follow the [Pico documentation](https://pico-docs.brevis.network/writing-apps/programs) and the [EVM Pico apps examples](https://github.com/brevis-network/evm-pico-apps) to write your program. Build it into an ELF with:

```
cd <app-name>
cargo pico build
cd ..
```

### Generate the Verification Key (App ID)

Every ELF has a deterministic verification key. Use the helper in `evm-pico-apps`:

```
VK_VERIFICATION=true cargo run -r --bin gen-app-id -- --elf <app-name>/elf/riscv32im-pico-zkvm-elf
```

Example (Fibonacci app):

```
VK_VERIFICATION=true cargo run -r --bin gen-app-id -- --elf fibonacci/elf/riscv32im-pico-zkvm-elf
```

Sample output:

```
Generated app_id: 0x00399db87f8d0d43e1795c4aebffe8cc58486e41b98371bdf667f3d29ce4476b
```

### Generate Input Data and Public Values Digest

Use the [example apps](https://github.com/brevis-network/evm-pico-apps?tab=readme-ov-file#example-apps) as a template. They typically emit both the encoded input data and the public values digest expected by the prover. Example (Fibonacci, `n=100`):

```
VK_VERIFICATION=true cargo run -r --bin gen-inputs-fibonacci -- --n 100
```

Sample output:

```
=== For Onchain ProofRequest ===
vk: 0x00399db87f8d0d43e1795c4aebffe8cc58486e41b98371bdf667f3d29ce4476b
publicValuesDigest: 0x12c6f9f81993158e5d1e480b643b0466160893ebb0531e8c3ad7dd22c3fdeaa3
inputData: 0x01000000000000000400000000000000640000000000000000000000
```

Persist these values—they flow directly into the proof-request transaction.

## Send a Proof Request

The easiest way to submit requests is via the [CLI utility](./tools.md#request-proofs), which batches multiple entries and handles encoding. For the Base Mainnet Beta Launch you must hold sufficient `USDC` to cover the proof fee.

To send a single request through a block explorer instead:

1. Approve USDC spending for `BrevisMarket` (address `0x64A364888eeafc0F72e7788DD2fBEc9a456b305e`) using the [USDC contract](https://basescan.org/token/0x833589fcd6edb6e08f4c7c32d4f71b54bda02913#writeProxyContract).
2. Call `requestProof` on [BrevisMarket](https://basescan.org/address/0x64A364888eeafc0F72e7788DD2fBEc9a456b305e#writeProxyContract) with the parameters gathered above.

Use the checklist below to fill each field correctly:

1. **`nonce`** – Unique identifier you choose per request.
2. **`vk`** – The verification key/app ID from [Generate the Verification Key](#generate-the-verification-key-app-id).
3. **`publicValuesDigest`** – Output from [Generate Input Data and Public Values Digest](#generate-input-data-and-public-values-digest).
4. **`imgURL`** – URL hosting the ELF artifact built in [Build the Application (ELF Artifact)](#build-the-application-elf-artifact).
5. **`inputData`** – Encoded input from the generator step.
6. **`inputURL`** – Optional URL pointing to the input payload when it is too large to inline. Provide either `inputData` or `inputURL` (or both).
7. **`fee` tuple (`maxFee`, `minStake`, `deadline`)**:
	- `maxFee`: Maximum amount (USDC on Base) you are willing to pay for the proof.
	- `minStake`: Minimum prover stake required to bid on this request.
	- `deadline`: Unix timestamp by which the proof must be submitted. The deadline must be within 30 days of the request time.

## Sample Apps for Testing

Use these reference programs when validating your setup:

- **[Fibonacci](https://github.com/brevis-network/evm-pico-apps/tree/main/fibonacci)** – Computes the *n*-th Fibonacci number from a `u32` input. Lightweight workload for quick smoke tests.
- **[Tendermint](https://github.com/brevis-network/pico/tree/main/examples/tendermint/app)** – Verifies consensus transitions between Tendermint light blocks. Medium-weight workload that exercises Merkle proofs and signature checks.
- **[Reth](https://github.com/brevis-network/pico/tree/main/perf/bench_apps/reth-pico)** – Executes Ethereum block verification via the Reth executor. Heavy workload representative of production proving jobs.

Build ELFs for these samples with `cargo pico build` inside each app directory, and use the matching input generators (where provided) to obtain the request payloads.

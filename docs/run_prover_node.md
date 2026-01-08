# Prover operation manual

This manual explains how to spin up a prover node for the Brevis Proving Network on Base mainnet. A prover node must run both the Pico proving service (to generate proofs) and the bidder service (to interact with the proving network).

## Table of contents

- [Up the pico proving service](#up-the-pico-proving-service)
   - [GPU Machine](#gpu-machine)
   - [CPU Machine](#cpu-machine)
- [Up the bidder service](#up-the-bidder-service)
   - [Prepare EC2 machine and install dependencies](#prepare-ec2-machine-and-install-dependencies)
   - [Setup binary, db, config and accounts](#setup-binary-db-config-and-accounts)
   - [Initialize Prover (StakingController)](#initialize-prover-stakingcontroller)
   - [Run the bidder node](#run-the-bidder-node)

## Up the pico proving service

A GPU host is strongly recommended. For small workloads or experimentation, a CPU host also works. The subsections below outline each setup.

### GPU Machine

1. Follow [multi-machine-setup.md](https://github.com/brevis-network/pico-ethproofs/blob/main/docs/multi-machine-setup.md) to prepare the GPU box.
2. Install [Docker](https://docs.docker.com/engine/install) and add your user to the `docker` group:
   ```bash
   sudo groupadd docker 2>/dev/null || true && sudo usermod -aG docker $USER
   ```
   If Docker reports `could not select device driver "" with capabilities: [[gpu]]`, install the [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) and restart Docker:
   ```bash
   sudo systemctl restart docker
   ```
3. Download the GPU Pico proving service image from `/home/ubuntu`:
   ```bash
   curl -sL -O https://pico-proofs.s3.us-west-2.amazonaws.com/prover-network/mainnet/pico_proving_service_gpu_v1_0_1.tar
   ```
4. Delete the old image from Docker and load the new image:
   If old image exists, delete it firstly
   ```bash
   docker rmi -f pico-proving-service-gpu:latest
   ```

   Load the new downloaded image
   ```bash
   docker load -i pico_proving_service_gpu_v1_0_1.tar
   docker tag pico-proving-service-gpu:v1.0.1 pico-proving-service-gpu:latest
   ```
5. Clone the repository and enter the GPU docker folder:
   ```bash
   # must switch to tag `v1.0.1` if you have already cloned pico-proving-service
   git clone --branch v1.0.1 https://github.com/brevis-network/pico-proving-service
   cd pico-proving-service/docker/gpu
   ```
6. Copy the environment template:
   ```bash
   cp .env.example .env
   ```
   - Fix `PROVER_COUNT` to the number of GPUs on your machine.
   - The `SPLIT_THRESHOLD`, `CHUNK_SIZE`, `MEM_POOL_RESERVE_SIZE` and `PICO_GPU_MEM` are set to default for RTX 5090. For RTX 4090, comment the settings for 5090 and enable the settings for 4090.
   Leave the others unless you are sure they need to change.
   If you encounter a GPU memory allocation issue, you could enable `MAX_EMULATION_CYCLES` to give a try, its value is machine specific.

7. Download dependencies and bring up the containers:
   ```bash
   # delete the old gnark files for upgrade if exist
   rm -rf ../gnark_downloads

   make download-gnark
   make up
   ```
8. Verify the containers:
   ```bash
   docker ps
   ```
   You should see `pico-proving-service` and `pico_gnark_server`. The Gnark server produces the final on-chain verifiable proof.
9. Review the `Makefile` for other targets (down/restart/clean). For logs, run:
   ```bash
   make logs-server
   make logs-gnark
   ```

### CPU Machine

1. Prepare the host:
   - Instance: AWS `r7i.16xlarge` (64 vCPUs) or equivalent.
   - OS: `ubuntu-24.04-amd64-server`.
   - Install prerequisites:
     - [Rust](https://www.rust-lang.org/tools/install) (restart the shell after installation).
     - Build tools: `sudo apt-get update && sudo apt-get install -y build-essential cmake git pkg-config libssl-dev protobuf-compiler`
     - [sqlx-cli](https://github.com/launchbadge/sqlx/tree/main/sqlx-cli): `cargo install sqlx-cli`
2. Install Docker and add your user to the `docker` group:
   ```bash
   sudo groupadd docker 2>/dev/null || true && sudo usermod -aG docker $USER
   ```
3. Download the CPU image from `/home/ubuntu`:
   ```bash
   curl -sL -O https://pico-proofs.s3.us-west-2.amazonaws.com/prover-network/mainnet/pico-proving-service-cpu.tar
   ```
4. Delete the old image from Docker and load the new image:
   If old image exists, delete it firstly
   ```bash
   docker rmi -f pico-proving-service-cpu:latest
   ```

   Load the new downloaded image
   ```bash
   docker load -i pico-proving-service-cpu.tar
   ```
5. Clone the repository and enter the CPU docker folder:
   ```bash
   git clone --branch v1.2.2 https://github.com/brevis-network/pico-proving-service
   cd pico-proving-service/docker/cpu
   ```
6. Copy the environment template:
   ```bash
   cp .env.example .env
   ```
   Keep the default values unless you have a specific reason to override them.
7. Download dependencies and start the containers:
   ```bash
   # delete the old gnark files for upgrade if exist
   rm -rf ../gnark_downloads

   make download-gnark
   make up
   ```
8. Check container status:
   ```bash
   docker ps
   ```
   You should see `pico-proving-service` and `pico_gnark_server`. The Gnark server produces the final on-chain verifiable proof.
9. Review the `Makefile` for lifecycle targets (stop/restart/clean). For logs, run:
   ```bash
   make logs-server
   make logs-gnark
   ```

## Up the bidder service

### Prepare EC2 machine and install dependencies

1. Launch an Ubuntu EC2 instance with appropriate security groups and a key pair.
2. Install Go (>=1.16):
   ```bash
   sudo snap install go --classic
   mkdir -p "$HOME/go/bin"
   ```
3. Install CockroachDB:
   ```bash
   curl -sL https://binaries.cockroachdb.com/cockroach-v24.2.3.linux-amd64.tgz \
     | sudo tar -xz --strip 1 -C /usr/local/bin cockroach-v24.2.3.linux-amd64/cockroach
   sudo chmod +x /usr/local/bin/cockroach
   ```
4. Configure CockroachDB as a systemd service:
   ```bash
   sudo mkdir -p /var/log/crdb
   sudo touch /var/log/crdb/out.log /var/log/crdb/err.log

   sudo tee /etc/systemd/system/crdb.service << EOF
   [Unit]
   Description=CockroachDB single node
   After=network-online.target

   [Service]
   WorkingDirectory=/home/ubuntu
   ExecStart=/usr/local/bin/cockroach start-single-node --insecure --listen-addr=localhost:26257 \
       --http-addr=localhost:18080 --store=path=/home/ubuntu/db
   StandardOutput=append:/var/log/crdb/out.log
   StandardError=append:/var/log/crdb/err.log
   Restart=always
   User=ubuntu
   Group=ubuntu
   RestartSec=10

   [Install]
   WantedBy=multi-user.target
   EOF

   sudo systemctl enable crdb.service
   sudo systemctl start crdb.service
   ```
5. Configure Go environment variables by appending the following to `$HOME/.profile`:
   ```bash
   export GOBIN=$HOME/go/bin
   export GOPATH=$HOME/go
   export PATH=$PATH:$GOBIN
   ```
   Then reload:
   ```bash
   source $HOME/.profile
   ```

### Setup binary, db, config and accounts

1. Clone the bidder service repo from `/home/ubuntu`:
   ```bash
   git clone https://github.com/brevis-network/prover-network-bidder
   cd prover-network-bidder
   git checkout main
   ```
2. Initialize the CockroachDB schema:
   ```bash
   cat $HOME/prover-network-bidder/dal/schema.sql | cockroach sql --insecure
   ```
3. Build and install the bidder binary:
   ```bash
   cd ./cmd/service
   go build -o bidder
   cp ./bidder $HOME/go/bin
   cd ~
   ```
4. Copy bidder configs:
   ```bash
   git clone https://github.com/brevis-network/prover-network-ops
   cp -a prover-network-ops/node-configs ~/.bidder
   ```
5. Edit `~/.bidder/config.toml`:

   You may use separate accounts for staking (`prover`) and for bidding/submitting proofs (`submitter`), or reuse the same Ethereum account for both.

   Because the prover and submitter roles are decoupled, you can keep the **prover** account in a hardware/HD wallet (or any setup you do not want online), and run the bidder service with a separate **submitter** account that is allowed to sign and submit proofs. In this setup, the prover account is used for staking and identity, while the submitter account is used operationally by the bidder.

   | Field | Description |
   | ----- | ----------- |
   | `prover_url` | Pico proving service gRPC endpoint (`${pico-ip}:${port}`, default port `50052`). |
   | `prover_eth_addr` | Ethereum address of the prover account. |
   | `submitter_keystore` | Path to the submitter keystore JSON (or AWS KMS reference). |
   | `submitter_passphrase` | Keystore password (or `apikey:apisec` for AWS KMS). |

   Optional tuning fields:

   | Field | Description |
   | ----- | ----------- |
   | `prover_gas_price` | Price per prove cycle (bid fee = `prove_cycles * prover_gas_price / 1e12`). Cycles are auto-computed; set the price per your economics. |
   | `prove_min_duration` | Skip requests whose remaining time (reveal → deadline) is less than this many seconds. |
   | `max_input_size` | `0` means no limit; otherwise skip requests with larger inputs. |
   | `max_fee` | Skip requests whose bid fee would exceed this ceiling. |
   | `vk_whitelist` | If empty, accept all requests. Otherwise process only VKs on this list. |
   | `vk_blacklist` | Skip requests targeting VKs on this list. |

   > Note: (1) Fees are denominated in the staking token. (2) A VK digest is generated when building the ELF and uniquely identifies a zk program.

### Initialize Prover (StakingController)

To join the proving network, initialize your prover on the [StakingController](https://basescan.org/address/0x9c0D8C5F10f0d3A02D04556a4499964a75DBf4A3#writeProxyContract). BREV is the staking token on Base mainnet (token address `0x086F405146Ce90135750Bbec9A063a8B20A8bfFb`). The CLI command [`tools init-prover`](./tools.md#init-prover) automates this, but you can also use a block explorer. Perform the first three steps with your **prover** account:

1. Approve the StakingController to spend your BREV: [BREV contract](https://basescan.org/token/0x086F405146Ce90135750Bbec9A063a8B20A8bfFb#writeProxyContract) → `approve(0x9c0D..., amount)`.
2. Call `initializeProver` on the [StakingController](https://basescan.org/address/0x9c0D8C5F10f0d3A02D04556a4499964a75DBf4A3#writeProxyContract) with a commission rate in basis points (e.g., 500 bps = 5%). Choose a commission rate that matches your policy. This call also transfers the minimum stake. At the present time, please make sure your prover account has at least 1000 BREV.

   Commission rates support both a default and per-source overrides via `setCommissionRate(source, rate)`:
   - Default commission rate: your global fallback rate, applied to any reward source without a specific override.
   - Per-source override: a rate specific to a given reward source (for example, you may set a higher rate for BrevisMarket rewards to cover GPU costs, while keeping a lower rate for other reward sources to remain competitive for stakers).
   
   Example operation: after `initializeProver`, set a 50% commission for BrevisMarket rewards by calling `setCommissionRate(brevisMarketAddr, 5000)` (50% = 5000 bps), where `brevisMarketAddr` is the BrevisMarket contract address.
3. Call `setProverProfile` to publish your metadata.
4. If the submitter uses a different account:
   - As the submitter, call `setSubmitterConsent` on [BrevisMarket](https://basescan.org/address/0xcCec2a9FE35b6B5F23bBF303A4e14e5895DeA127#writeProxyContract).
   - As the prover, call `registerSubmitter` on the same contract.

### Run the bidder node

1. Create a systemd service:
   ```bash
   sudo mkdir -p /var/log/bidder
   sudo touch /var/log/bidder/app.log

   sudo tee /etc/systemd/system/bidder.service << EOF
   [Unit]
   Description=bidder daemon
   After=network-online.target

   [Service]
   Environment=HOME=/home/ubuntu
   ExecStart=/home/ubuntu/go/bin/bidder --config /home/ubuntu/.bidder/config.toml
   StandardOutput=append:/var/log/bidder/app.log
   StandardError=append:/var/log/bidder/app.log
   Restart=always
   RestartSec=3
   User=ubuntu
   Group=ubuntu
   LimitNOFILE=4096

   [Install]
   WantedBy=multi-user.target
   EOF
   ```
2. Create `/etc/logrotate.d/bidder` and add the following:

   ```
   /var/log/bidder/*.log {
       compress
       copytruncate
       daily
       maxsize 30M
       rotate 30
   }
   ```
3. Enable and start the service:
   ```bash
   sudo systemctl enable bidder
   sudo systemctl start bidder
   ```

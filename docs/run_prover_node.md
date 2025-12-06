# Prover operation manual

This manual explains how to spin up a prover node for the Brevis Proving Network. A prover node must run both the Pico proving service (to generate proofs) and the bidder service (to interact with the proving network).

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
   curl -sL -O https://pico-proofs.s3.us-west-2.amazonaws.com/prover-network/pico-proving-service-gpu.tar
   ```
4. Load the image into Docker:
   ```bash
   docker load -i pico-proving-service-gpu.tar
   ```
5. Clone the repository and enter the GPU docker folder:
   ```bash
   git clone https://github.com/brevis-network/pico-proving-service
   cd pico-proving-service/docker/gpu
   ```
6. Copy the environment template:
   ```bash
   cp .env.example .env
   ```
   Leave the defaults unless you are sure they need to change.
7. Download dependencies and bring up the containers:
   ```bash
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
     - Build tools: `sudo apt-get install -y build-essential cmake git pkg-config libssl-dev`
     - [sqlx-cli](https://github.com/launchbadge/sqlx/tree/main/sqlx-cli): `cargo install sqlx-cli`
2. Install Docker and add your user to the `docker` group:
   ```bash
   sudo groupadd docker 2>/dev/null || true && sudo usermod -aG docker $USER
   ```
3. Download the CPU image from `/home/ubuntu`:
   ```bash
   curl -sL -O https://pico-proofs.s3.us-west-2.amazonaws.com/prover-network/pico-proving-service-cpu.tar
   ```
4. Load the image:
   ```bash
   docker load -i pico-proving-service-cpu.tar
   ```
5. Clone the repository and enter the CPU docker folder:
   ```bash
   git clone https://github.com/brevis-network/pico-proving-service
   cd pico-proving-service/docker/cpu
   ```
6. Copy the environment template:
   ```bash
   cp .env.example .env
   ```
   Keep the default values unless you have a specific reason to override them.
7. Download dependencies and start the containers:
   ```bash
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

   | Field | Description |
   | ----- | ----------- |
   | `prover_url` | Pico proving service gRPC endpoint (`${pico-ip}:${port}`, default port `50052`). |
   | `prover_eth_addr` | Ethereum address of the prover account. |
   | `submitter_keystore` | Path to the submitter keystore JSON (or AWS KMS reference). |
   | `submitter_passphrase` | Keystore password (or `apikey:apisec` for AWS KMS). |

   Optional tuning fields:

   | Field | Description |
   | ----- | ----------- |
   | `prover_gas_price` | Price per prove cycle (bid fee = `prove_cycles * prover_gas_price / 1e9`). Cycles are auto-computed; set the price per your economics. |
   | `prove_min_duration` | Skip requests whose remaining time (reveal → deadline) is less than this many seconds. |
   | `max_input_size` | `0` means no limit; otherwise skip requests with larger inputs. |
   | `max_fee` | Skip requests whose bid fee would exceed this ceiling. |
   | `vk_whitelist` | If empty, accept all requests. Otherwise process only VKs on this list. |
   | `vk_blacklist` | Skip requests targeting VKs on this list. |

   > Note: (1) Fees are denominated in the staking token. (2) A VK digest is generated when building the ELF and uniquely identifies a zk program.

### Staking as a bidder

To join the proving network as a bidder, you must initialize yourself as a prover in the [StakingController](https://basescan.org/address/0x435f3Ee9673d6a1c73AddD8F5B6bF643E882E0B3#writeProxyContract). USDC is the staking token during the current mainnet beta; the official mainnet launch will use the BREV token for staking. The CLI command [`tools init-prover`](./tools.md#init-prover) automates this, but you can also use a block explorer. Perform the first three steps with your **prover** account:

1. Approve the StakingController to spend your USDC: [USDC contract](https://basescan.org/token/0x833589fcd6edb6e08f4c7c32d4f71b54bda02913#writeProxyContract) → `approve(0x435f..., amount)`.
2. Call `initializeProver` on the [StakingController](https://basescan.org/address/0x435f3Ee9673d6a1c73AddD8F5B6bF643E882E0B3#writeProxyContract) with a commission rate 10000 bps (100%). The current mainnet beta is limited to prover-operated staking, so keeping the commission at 100% prevents losing collected fees. This call also transfers the minimum stake.
3. Call `setProverProfile` to publish your metadata.
4. If the submitter uses a different account:
   - As the submitter, call `setSubmitterConsent` on [BrevisMarket](https://basescan.org/token/0x64A364888eeafc0F72e7788DD2fBEc9a456b305e#writeProxyContract).
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

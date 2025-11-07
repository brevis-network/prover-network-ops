# Bidder operation manual

This manual describes the process of spinning up a bidder node of Brevis Proving Network. A bidder node needs to run a pico proving service to prove a request, and a bidder service to interact with the proving network.

## Up the pico proving service

1. Select [Machine and OS](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#machine-and-os)

2. Install required [Prerequisites](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#prerequisites)

3. Initialize [Local DB](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#local-db-initialization)

4. Start [service](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#service-start)

### Run pico proving service as a system service

If you need to run the pico proving service as a system service, shut down the service in above step 3 and continue the follow steps:

1. Under pico-proving-service folder (the repo you cloned), copy release binary and DB to a persistent folder

    ```sh
    mkdir $HOME/.pico
    cp ./target/release/server $HOME/.pico/server
    cp ./pico_proving_service.db $HOME/.pico/pico_proving_service.db
    ```

2. Select an appropriate `MAX_EMULATION_CYCLES` value for the input tasks supported by the prover. If unset, all proving tasks are supported.

  For CPU machine, reference the benchmark result on `r7i.16xlarge`:
  ```
  - Fibonacci n = 1_000_000
  Cycles: 12001916
  Proving time: 64.313s


  - Fibonacci n = 10_000_000
  Cycles: 120001916
  Proving time: 394.217s

  - Reth block_number = 18884864
  Cycles: 90169715
  Proving time: 385.187s
  ETH gas: 4,266,500

  - Reth block_number = 17106222
  Cycles: 176733255
  Proving time: 733.075s
  ETH gas: 10,781,405
  ```

  For GPU machine, reference the benchmark result on `8 X NVIDIA RTX 5090`:
  ```
  - Fibonacci n = 1_000_000
  Cycles: 12001916
  Proving time: 8.345s


  - Fibonacci n = 10_000_000
  Cycles: 120001916
  Proving time: 15.025s

  - Reth block_number = 18884864
  Cycles: 90169715
  Proving time: 22.03s
  ETH gas: 4,266,500

  - Reth block_number = 17106222
  Cycles: 176733255
  Proving time: 32.024s
  ETH gas: 10,781,405
  ```

3. Execute below to configure pico as a system service (assume `$Home=/home/ubuntu`, if not, please replace `/home/ubuntu` to your real one)

    ```sh
    sudo mkdir -p /var/log/pico
    sudo touch /var/log/pico/app.log

    sudo tee /etc/systemd/system/pico.service << EOF
    [Unit]
    Description=Pico Proving Service
    After=network-online.target

    [Service]
    WorkingDirectory=/home/ubuntu/.pico
    Environment=DATABASE_URL=sqlite:///home/ubuntu/.pico/pico_proving_service.db?mode=rwc
    Environment=RUST_LOG=debug
    Environment=NUM_THREADS=8
    Environment=RUSTFLAGS="-C target-cpu=native -C target-feature=+avx512f,+avx512ifma,+avx512vl"
    Environment=JEMALLOC_SYS_WITH_MALLOC_CONF="retain:true,background_thread:true,metadata_thp:always,dirty_decay_ms:-1,muzzy_decay_ms:-1,abort_conf:true"
    Environment=CHUNK_SIZE=2097152
    Environment=CHUNK_BATCH_SIZE=32
    Environment=SPLIT_THRESHOLD=1048576
    Environment=PROVER_COUNT=32
    Environment=RUST_MIN_STACK=16777216
    Environment=VK_VERIFICATION=false
    # Environment=MAX_EMULATION_CYCLES=200000000 # optional
    ExecStart=/home/ubuntu/.pico/server
    StandardOutput=append:/var/log/pico/app.log
    StandardError=append:/var/log/pico/app.log
    Restart=always
    User=ubuntu
    Group=ubuntu
    RestartSec=3

    [Install]
    WantedBy=multi-user.target
    EOF
    ```

4. Create `/etc/logrotate.d/pico` and add the following:

    ```
    /var/log/pico/*.log {
        compress
        copytruncate
        daily
        maxsize 30M
        rotate 30
    }
    ```

5. Enable and start the service:

    ```sh
    ulimit -s unlimited
    sudo systemctl enable pico
    sudo systemctl start pico
    ```

## Up the bidder service

### Prepare EC2 machine and install dependencies 

1. Start an EC2 machine with the Ubuntu 20.04 LTS image. Use the appropriate security groups and a keypair that you have access to.

2. Install go (at least 1.16):

    ```sh
    sudo snap install go --classic
    mkdir -p $HOME/go/bin
    ```

3. Install CockroachDB:

    ```sh
    curl -sL https://binaries.cockroachdb.com/cockroach-v21.1.3.linux-amd64.tgz | sudo tar -xz --strip 1 -C /usr/local/bin cockroach-v21.1.3.linux-amd64/cockroach
    sudo chmod +x /usr/local/bin/cockroach
    ```

4. Execute below to config crdb as system service

    ```sh
    sudo mkdir -p /var/log/crdb
    sudo touch /var/log/crdb/out.log
    sudo touch /var/log/crdb/err.log

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

5. Set \$GOBIN and add \$GOBIN to \$PATH. Edit `$HOME/.profile` and add:

    ```sh
    export GOBIN=$HOME/go/bin; export GOPATH=$HOME/go; export PATH=$PATH:$GOBIN
    ```

    to the end, then:

    ```sh
    source $HOME/.profile
    ```

### Setup binary, db, config and accounts

1. From the `/home/ubuntu` directory, clone the `prover-network-bidder` repository

    ```sh
    git clone https://github.com/brevis-network/prover-network-bidder
    cd prover-network-bidder
    git checkout main
    ```

2. Initialize the db

    ```sh
    cat $HOME/prover-network-bidder/dal/schema.sql | cockroach sql --insecure
    ```

3. Install the bidder binary

    ```sh
    cd ./cmd/service
    go build -o bidder
    cp ./bidder $HOME/go/bin
    cd ~
    ```

4. From the `/home/ubuntu` directory, clone the `prover-network-bidder-ops` repository, then copy the config files

    ```sh
    git clone https://github.com/brevis-network/prover-network-bidder-ops
    cp -a prover-network-bidder-ops/node-configs ~/.bidder
    ```

5. Make sure the fields in `~/.bidder/config.toml` have the correct values:

    A bidder can use different accounts to `stake` and `bid & submit proof` seprately. The former we call it `prover` account and later `submitter` account. You can use a same ETH account for both of them.
    
    | Field | Description |
    | ----- | ----------- |
    | prover_url | the pico proving service grpc endpoint in format `${pico machine ip}:${port}`. the service defaultly starts at port 50052 |
    | prover_eth_addr | The Ethereum address of the prover |
    | submitter_keystore | The path to your prepared submitter ethereum keystore json (or use AWS KMS) |
    | submitter_passphrase | The passphrase to the submitter keystore (or apikey:apisec if using AWS KMS) |

    Update the below fields on demand in accordance with your requirement:
    | Field | Description |
    | ----- | ----------- |
    | prover_gas_price | the price of a prove cycle. the bid fee to a request comes from `prove cycles * prover gas price`. the prove cycles of a request is auto calculated by pico service, while the prover gas price can be set in consideration of your business |
    | prove_min_duration | skip the requests that the duration from proving start time (right after reveal phase) to deadline is less than the `prove_min_duration` |
    | max_input_size | default 0 means no limit. if this value is non-zero, and request input is larger, skip request |
    | max_fee | skip the requests that the calculated bid fee exceeds the `max_fee` |

	Note, the fee is denominated in staking token.

### Staking as a bidder

To join the proving network as a bidder, you must stake staking token in [StakingController](https://sepolia.arbiscan.io/address/0x4eE8ec243dceC0a6A5676470d4dBfA71CE96F069#writeProxyContract). 

Please operate below steps using your `Prover` account.

1. Firstly, use [explorer Faucet](https://sepolia.arbiscan.io/address/0x9C4e124141A599482b08492a03c49e26CCA21bAA#writeContract) to get `drip` some [testnet staking token](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074)

2. Use [explorer StakingToken](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074#writeContract) to `approve` StakingController 0x4eE8ec243dceC0a6A5676470d4dBfA71CE96F069 to spend your staking token

3. Use [explorer StakingController](https://sepolia.arbiscan.io/address/0x4eE8ec243dceC0a6A5676470d4dBfA71CE96F069#writeProxyContract) to `initializeProver` with a default commission rate. It will transfer a configured minimum staking amount from your wallet to `StakingController`

4. Use [explorer StakingController](https://sepolia.arbiscan.io/address/0x4eE8ec243dceC0a6A5676470d4dBfA71CE96F069#writeProxyContract) to `setProverProfile`

5. Use [explorer StakingController](https://sepolia.arbiscan.io/address/0x4eE8ec243dceC0a6A5676470d4dBfA71CE96F069#writeProxyContract) to `stake` more as you wish

6. If you use different account for submitter, please:
 * Firstly as a submitter, use [explorer BrevisMarket](https://sepolia.arbiscan.io/address/0x9c19d2De433217FB4b41a5D8d35aB8eE4A7b0DFa#writeProxyContract) to `setSubmitterConsent` (submitter grants consent).
 * And then as a prover, use [explorer BrevisMarket](https://sepolia.arbiscan.io/address/0x9c19d2De433217FB4b41a5D8d35aB8eE4A7b0DFa#writeProxyContract) to `registerSubmitter` (prover registers the submitter).

You can also use below tool to do the job.

#### A tool to stake as a bidder

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `stake_config.toml` to fill in:

    | Field | Description |
    | ----- | ----------- |
    | prover_keystore | The path to your prover ethereum account keystore json |
    | prover_passphrase | The passphrase to the prover keystore |
    | submitter_keystore | Fill in if you need a differnt account as the submitter |
    | submitter_passphrase | Fill in if you need a differnt account as the submitter |
    | prover_name | a name to identify you or your organization |
    | prover_icon | the url of the icon that represents you or your organization |
    | staking_amt | A non-zero value if you want to stake more in addition to `minSelfStake` |
    | commission_rate_bps | commission rate for delegators who add staking on you |

3. Execute below command to send the requsets:

    ```
    ./tools stake --config ./stake_config.toml --init true
    ```

### Run the bidder node

1. Prepare the bidder system service:

    ```sh
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

    ```sh
    sudo systemctl enable bidder
    sudo systemctl start bidder
    ```

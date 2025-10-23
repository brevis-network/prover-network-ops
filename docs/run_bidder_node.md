# Bidder operation manual

This manual describes the process of spinning up a bidder node of Brevis Proving Network. A bidder node needs to run a pico proving service to prove a request, and a bidder service to interact with the proving network.

## Up the pico proving service

1. Install required [Prerequisites](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#prerequisites)

2. Initialize [Local DB](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#local-db-initialization)

3. Start [service](https://github.com/brevis-network/pico-proving-service?tab=readme-ov-file#service-start)

### Run pico proving service as a system service

If you need to run the pico proving service as a system service, shut down the service in above step 3 and continue the follow steps:

1. Under pico-proving-service folder (the repo you cloned), copy release binary and DB to a persistent folder

    ```sh
    mkdir $HOME/.pico
    cp ./target/release/server $HOME/.pico/server
    cp ./pico_proving_service.db $HOME/.pico/pico_proving_service.db
    ```

2. Execute below to configure pico as a system service (assume `$Home=/home/ubuntu`, if not, please replace `/home/ubuntu` to your real one)

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

3. Enable and start the service:

    ```sh
    sudo systemctl enable pico
    sudo systemctl start pico
    ```

## Up the bidder service

### Prepare EC2 machine and install dependencies 

1. Start an EC2 machine with the Ubuntu 20.04 LTS image. Use the appropriate security groups and a keypair that you have access to.

2. Install go (at least 1.16):

    ```sh
    sudo snap install go --classic
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
    WorkingDirectory=$HOME
    ExecStart=/usr/local/bin/cockroach start-single-node --insecure --listen-addr=localhost:26257 \
      --http-addr=localhost:18080 --store=path=$HOME/db
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
    git checkout main
    ```

2. Initialize the db

    ```sh
    cat $HOME/prover-network-bidder/dal/schema.sql | cockroach sql --insecure
    ```

3. Install the bidder binary

    ```sh
    cd prover-network-bidder/cmd/service
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

    | Field | Description |
    | ----- | ----------- |
    | prover_url | the pico proving service grpc endpoint |
    | bidder_keystore | The path to your prepared ethereum keystore json (or use AWS KMS) |
    | bidder_passphrase | The passphrase to the bidder keystore (or apikey:apisec if using AWS KMS) |
    | bidder_eth_addr | The Ethereum address of the bidder |

### Staking as a bidder

To join the proving network as a bidder, you must stake staking token in [StakingController](https://sepolia.arbiscan.io/address/0x8B83b9808DE79D5EEE97417bB14f82c41bCcD6F0#writeProxyContract). 

1. Firstly, use [explorer Faucet](https://sepolia.arbiscan.io/address/0x9C4e124141A599482b08492a03c49e26CCA21bAA#writeContract) to get `drip` some [testnet staking token](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074)

2. Use [explorer StakingToken](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074#writeContract) to `approve` StakingController 0x8B83b9808DE79D5EEE97417bB14f82c41bCcD6F0 to spend your staking token

3. Use [explorer StakingController](https://sepolia.arbiscan.io/address/0x8B83b9808DE79D5EEE97417bB14f82c41bCcD6F0#writeProxyContract) to `initializeProver` with a default commission rate. It will transfer a configured minimum staking amount from your wallet to `StakingController`

4. Use [explorer StakingController](https://sepolia.arbiscan.io/address/0x8B83b9808DE79D5EEE97417bB14f82c41bCcD6F0#writeProxyContract) to `stake` more as you wish

### Run the bidder node

1. Prepare the bidder system service:

    ```sh
    sudo mkdir -p /var/log/bidder
    sudo touch /var/log/bidder/app.log
    sudo touch /etc/systemd/system/bidder.service
    ```

    Add the following to `/etc/systemd/system/bidder.service`:

    ```
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

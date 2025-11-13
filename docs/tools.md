# Tools instruction

Brevis provides tool utilities to facilitate the onchain operations. In Details, for a prover to 
- [Init prover](#init-prover)
- [Claim commission](#claim-commission)
- [Stake more](#stake)
- [Unstake](#unstake)

and for a user to
- [Request proofs](#request-proofs)
- [Refund not fulfilled requests](#refund-not-fulfilled-requests)
- [Stake to a prover](#stake)
- [Unstake from a prover](#unstake)

## Init prover

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | The path to your prover ethereum account keystore json |
    | chain | passphrase | The passphrase to the prover keystore |
    | init_prover | submitter_keystore | Fill in if you need a differnt account as the submitter |
    | init_prover | submitter_passphrase | Fill in if you need a differnt account as the submitter |
    | init_prover | prover_name | a name to identify you or your organization |
    | init_prover | prover_icon | the url of the icon that represents you or your organization |

3. Execute below command:

    ```
    ./tools init-prover --config ./config.toml
    ```

## Claim commission

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | The path to your prover ethereum account keystore json |
    | chain | passphrase | The passphrase to the prover keystore |

3. Execute below command:

    ```
    ./tools claim-commission --config ./config.toml
    ```

## Stake

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | The path to your ethereum account keystore json |
    | chain | passphrase | The passphrase to the keystore |
    | stake | stake_to_prover | The prover address that you want to stake your token to (a prover can also use this command stake more to yourself) |
    | stake | stake_amt | The stake amount in Wei unit |

3. Execute below command:

    ```
    ./tools stake --config ./config.toml
    ```

## Unstake

For unstake, it takes two stages to complete. You should firstly send an unstake request, and then after a delay period, send another request to compeltely unstake.

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | The path to your ethereum account keystore json |
    | chain | passphrase | The passphrase to the keystore |
    | unstake | unstake_from_prover | The prover address that you want to unstake from (a prover can also use this command to unstake from the system and deactivate the prover role) |

3. Execute below command:

    - for request stage

    ```
    ./tools unstake --config ./config.toml --stage request
    ```

    - for complate stage

    ```
    ./tools unstake --config ./config.toml --stage complete
    ```

## Request proofs

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in below fields in `[chain]` section:

    | Field | Description |
    | ----- | ----------- |
    | keystore | The path to your ethereum account keystore json |
    | passphrase | The passphrase to the keystore |

Update the param values in `[[request]]` section accordingly. Provide multile `[[request]]` sections to send multiple requests.

3. Execute below command:

    ```
    ./tools request-proof --config ./config.toml
    ```

## Refund not fulfilled requests

1. Under `tools` folder in this repo, run `go build` to build the tool.

    ```
    cd tools
    go build
    ```

2. Then update the `config.toml` to fill in:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | The path to your ethereum account keystore json |
    | chain | passphrase | The passphrase to the keystore |
    | refund | req_ids | You can either fill in the request ids in array or leave this filed as empty array, and pass in `all` flag in the command |

3. Execute below command:

    - provide req_ids manually in `config.toml`

    ```
    ./tools request-proof --config ./config.toml
    ```

    - refund all refundable requests

    ```
    ./tools request-proof --config ./config.toml --all
    ```

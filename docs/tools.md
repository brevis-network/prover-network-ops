# Tools instruction

Brevis provides CLI utilities that streamline on-chain operations for the Base mainnet launch. Provers can:
- [Init prover](#init-prover), note that this step will also auto stake a configured minimum amount to ensure prover meets the minimum self-stake requirement. Prover can use [`stake`](#stake) command to stake more on demand.
- [Claim commission](#claim-commission)
- [Stake more](#stake)
- [Unstake](#unstake)

Users can:
- [Request proofs](#request-proofs)
- [Refund not fulfilled requests](#refund-not-fulfilled-requests)
- [Stake to a prover](#stake)
- [Unstake from a prover](#unstake)

## Init prover

1. From the `tools` directory in this repo, build the binary:

    ```
    cd tools
    go build
    ```

2. Update `config.toml` with the following fields:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | Path to your prover Ethereum account keystore JSON |
    | chain | passphrase | Passphrase for the prover keystore |
    | init_prover | submitter_keystore | (Optional) Submitter keystore if using a different account |
    | init_prover | submitter_passphrase | (Optional) Passphrase for the submitter account |
    | init_prover | prover_name | Name that identifies you or your organization |
    | init_prover | prover_icon | URL of the icon that represents you or your organization |

3. Run:

    ```
    ./tools init-prover --config ./config.toml
    ```

## Claim commission

1. From the `tools` directory, build the binary:

    ```
    cd tools
    go build
    ```

2. Update `config.toml` with:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | Path to your prover Ethereum account keystore JSON |
    | chain | passphrase | Passphrase for the prover keystore |

3. Run:

    ```
    ./tools claim-commission --config ./config.toml
    ```

## Stake

1. From the `tools` directory, build the binary:

    ```
    cd tools
    go build
    ```

2. Update `config.toml` with:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | Path to your Ethereum account keystore JSON |
    | chain | passphrase | Passphrase for the keystore |
    | stake | stake_to_prover | Prover address you want to stake to (provers can also stake more to themselves) |
    | stake | stake_amt | Stake amount in Wei |

3. Run:

    ```
    ./tools stake --config ./config.toml
    ```

## Unstake

Unstaking happens in two stages: submit the request, wait for the delay period, then complete the withdrawal.

1. From the `tools` directory, build the binary:

    ```
    cd tools
    go build
    ```

2. Update `config.toml` with:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | Path to your Ethereum account keystore JSON |
    | chain | passphrase | Passphrase for the keystore |
    | unstake | unstake_from_prover | Prover address you want to unstake from (provers can also use this to deactivate themselves) |

3. Run the commands for each stage:

    - Request stage

    ```
    ./tools unstake --config ./config.toml --stage request
    ```

    - Complete stage

    ```
    ./tools unstake --config ./config.toml --stage complete
    ```

## Request proofs

1. From the `tools` directory, build the binary:

    ```
    cd tools
    go build
    ```

2. Update the `[chain]` section in `config.toml`:

    | Field | Description |
    | ----- | ----------- |
    | keystore | Path to your Ethereum account keystore JSON |
    | passphrase | Passphrase for the keystore |

   Adjust the parameters inside each `[[request]]` section as needed. Add multiple `[[request]]` sections to submit more than one proof request.

3. Run:

    ```
    ./tools request-proof --config ./config.toml
    ```

## Refund not fulfilled requests

1. From the `tools` directory, build the binary:

    ```
    cd tools
    go build
    ```

2. Update `config.toml` with:

    | Section | Field | Description |
    | ------- | ----- | ----------- |
    | chain | keystore | Path to your Ethereum account keystore JSON |
    | chain | passphrase | Passphrase for the keystore |
    | refund | req_ids | Provide specific request IDs, or leave the array empty and use the `--all` flag |

3. Run the appropriate command:

    - If `req_ids` are specified in `config.toml`:

    ```
    ./tools request-proof --config ./config.toml
    ```

    - To refund every eligible request:

    ```
    ./tools request-proof --config ./config.toml --all
    ```

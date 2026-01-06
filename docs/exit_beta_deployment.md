# How to exit ProverNet (beta deployment sunset)

The ProverNet beta deployment has been sunset. If you previously participated as a prover on the beta deployment, follow the steps below to exit and withdraw your stake.

To exit ProverNet as a prover, you typically only need to do two things:

1. [Unstake](./tools.md#unstake)
	- This unstakes **all** of your stake/shares from the prover.
	- Unstaking happens in two stages: **request** first, then **complete** after the unstake delay period configured in the `StakingController` contract.

2. [Claim commission](./tools.md#claim-commission)

If you prefer not to use the CLI tools, you can perform the same actions via the BaseScan explorer contract UI.

Config templates:
- Beta deployment (sunset): `../tools/beta/config.toml`
- Mainnet deployment (current): `../tools/config.toml`
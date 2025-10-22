# How to request proof

Use [explorer BrevisMarket](https://sepolia.arbiscan.io/address/0x7c968e3b1FaE6599958104cbf40d17A4ba0c1d43#writeProxyContract) to request proof. Follow below instructions to obtain each param of a request.

1. nonce
A unique num to identify a request from you

2. vk
[!TODO]

3. publicValuesDigest
[!TODO]

4. imgURL
[!TODO]

5. inputData
[!TODO], input 0x if inputURL is provided

6. inputURL
[!TODO]

7. fee
Max fee in staking token (for testnet, it's [testnet staking token](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074) you wish to pay for the request. 

Note, Before initiate a proof request, you need to use [explorer StakingToken](https://sepolia.arbiscan.io/address/0x46b07178907650afc855763a8f83e65afec24074#writeContract) to approve `BrevisMarket` 0x7c968e3b1FaE6599958104cbf40d17A4ba0c1d43 to spend your staking token.

Please contact Brevis team for the faucet of the staking token.
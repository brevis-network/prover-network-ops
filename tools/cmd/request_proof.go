/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"
	"tools/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Request struct {
	Nonce              uint64 `mapstructure:"nonce"`
	Vk                 string `mapstructure:"vk"`
	PublicValuesDigest string `mapstructure:"public_value_digest"`
	ImgUrl             string `mapstructure:"img_url"`
	InputUrl           string `mapstructure:"input_url"`
	InputData          string `mapstructure:"input_data"`
	MaxFee             string `mapstructure:"max_fee"`
	MinStake           string `mapstructure:"min_stake"`
	Deadline           uint64 `mapstructure:"deadline"`
}

type Requests []*Request

func RequestProofCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-proof",
		Short: "interact with BrevisMarket contract to initiate a proof request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return requestProof()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.MarkFlagRequired(FlagConfig)
	return cmd
}

func init() {
	rootCmd.AddCommand(RequestProofCmd())
}

func requestProof() error {
	viper.SetConfigFile(config)
	err := viper.ReadInConfig()
	chkErr(err, "ReadInConfig")

	var c ChainConfig
	err = viper.UnmarshalKey("chain", &c)
	chkErr(err, "UnmarshalKey")

	ec, err := ethclient.Dial(c.ChainRpc)
	chkErr(err, "Dial")
	chid, err := ec.ChainID(context.Background())
	chkErr(err, "ChainID")
	if chid.Uint64() != c.ChainID {
		return fmt.Errorf("chainid mismatch! cfg has %d but onchain has %d", c.ChainID, chid.Uint64())
	}

	var reqs Requests
	err = viper.UnmarshalKey("request", &reqs)
	chkErr(err, "UnmarshalKey")
	if len(reqs) == 0 {
		return fmt.Errorf("should provide at least one request")
	}

	for i, r := range reqs {
		if (r.InputData == "0x" || r.InputData == "") && r.InputUrl == "" {
			return fmt.Errorf("req %d: should provide either input_data or input_url", i+1)
		}

		_, success := big.NewInt(0).SetString(r.MaxFee, 0)
		if !success {
			return fmt.Errorf("req %d: max_fee is not valid", i+1)
		}
		_, success = big.NewInt(0).SetString(r.MinStake, 0)
		if !success {
			return fmt.Errorf("req %d: min_stake is not valid", i+1)
		}
		if r.Deadline <= uint64(time.Now().Unix()) {
			return fmt.Errorf("req %d: deadline should be a future time", i+1)
		}
	}

	auth, _, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "CreateTransactOpts")
	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	brevisMarket, err := bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
	chkErr(err, "NewBrevisMarket")

	for i, r := range reqs {
		feeInt, _ := big.NewInt(0).SetString(r.MaxFee, 0)
		minStakeInt, _ := big.NewInt(0).SetString(r.MinStake, 0)
		tx, err := stakingToken.Approve(auth, common.HexToAddress(c.BrevisMarketAddr), feeInt)
		chkErr(err, fmt.Sprintf("req %d: Approve", i+1))
		log.Printf("req %d: approve tx: %s", i+1, tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, fmt.Sprintf("req %d: waitmined", i+1))
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalf("req %d: approve tx status is not success", i+1)
		}

		tx, err = brevisMarket.RequestProof(auth, bindings.IBrevisMarketProofRequest{
			Nonce:              r.Nonce,
			Vk:                 common.HexToHash(r.Vk),
			PublicValuesDigest: common.HexToHash(r.PublicValuesDigest),
			ImgURL:             r.ImgUrl,
			InputData:          common.FromHex(r.InputData),
			InputURL:           r.InputUrl,
			Fee: bindings.IBrevisMarketFeeParams{
				MaxFee:   feeInt,
				MinStake: minStakeInt,
				Deadline: r.Deadline,
			},
		})
		checkBrevisCustomError(err, fmt.Sprintf("req %d: RequestProof", i+1), bindings.IBrevisMarketABI)
		log.Printf("req %d: RequestProof tx: %s", i+1, tx.Hash())
		receipt, err = bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, fmt.Sprintf("req %d: waitmined", i+1))
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalf("req %d: RequestProof tx status is not success", i+1)
		}

		req, err := brevisMarket.ParseNewRequest(*receipt.Logs[1])
		chkErr(err, fmt.Sprintf("req %d: ParseNewRequest", i+1))
		log.Printf("req %d: reqId is %s", i+1, common.Bytes2Hex(req.Reqid[:]))
	}

	return nil
}

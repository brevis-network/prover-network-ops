/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"tools/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type UnstakeConfig struct {
	Prover string `mapstructure:"unstake_from_prover"`
}

func UnstakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake",
		Short: "init/complete unstake from a prover",
		RunE: func(cmd *cobra.Command, args []string) error {
			return unstake()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.Flags().StringVar(&stage, FlagStage, "request", "request or complete")
	cmd.MarkFlagRequired(FlagConfig)
	cmd.MarkFlagRequired(FlagStage)
	return cmd
}

const (
	FlagStage = "stage"
)

var (
	stage string
)

func init() {
	rootCmd.AddCommand(UnstakeCmd())
}

func unstake() error {
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

	var s UnstakeConfig
	err = viper.UnmarshalKey("unstake", &s)
	chkErr(err, "UnmarshalKey")

	auth, sender, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "prover CreateTransactOpts")

	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	if stage != "request" && stage != "complete" {
		log.Fatalln("stage param only accepts value `request` or `complete`")
	}

	if stage == "request" {
		shares, err := stakingController.GetStakeInfo(nil, common.HexToAddress(s.Prover), sender)
		chkErr(err, "GetStakeInfo")

		if shares.Sign() != 1 {
			log.Fatalf("no shares staked: prover %s, staker %s", s.Prover, sender.Hex())
		}

		tx, err := stakingController.RequestUnstake(auth, common.HexToAddress(s.Prover), shares)
		checkBrevisCustomError(err, "RequestUnstake", bindings.IStakingControllerABI)
		log.Printf("RequestUnstake tx: %s", tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("RequestUnstake tx status is not success")
		}
	} else {
		tx, err := stakingController.CompleteUnstake(auth, common.HexToAddress(s.Prover))
		checkBrevisCustomError(err, "CompleteUnstake", bindings.IStakingControllerABI)
		log.Printf("CompleteUnstake tx: %s", tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("CompleteUnstake tx status is not success")
		}
	}

	return nil
}

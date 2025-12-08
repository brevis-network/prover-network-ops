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

type StakeConfig struct {
	Prover   string `mapstructure:"stake_to_prover"`
	StakeAmt string `mapstructure:"stake_amt"`
}

func StakeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "stake to a prover",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stake()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.MarkFlagRequired(FlagConfig)
	return cmd
}

func init() {
	rootCmd.AddCommand(StakeCmd())
}

func stake() error {
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

	var s StakeConfig
	err = viper.UnmarshalKey("stake", &s)
	chkErr(err, "UnmarshalKey")

	auth, _, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "prover CreateTransactOpts")

	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	stakeAmt, success := big.NewInt(0).SetString(s.StakeAmt, 0)
	if !success {
		log.Fatalln("stake_amt is not a valid number")
	}

	if stakeAmt.Sign() == 0 {
		log.Fatalln("stake_amt should be larger than 0")
	}

	tx, err := stakingToken.Approve(auth, common.HexToAddress(c.StakingControllerAddr), stakeAmt)
	chkErr(err, "Approve")
	log.Printf("approve tx: %s", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("Approve tx status is not success")
	}
	time.Sleep(1 * time.Second)

	tx, err = stakingController.Stake(auth, common.HexToAddress(s.Prover), stakeAmt)
	checkBrevisCustomError(err, "Stake", bindings.IStakingControllerABI)
	log.Printf("Stake tx: %s", tx.Hash())
	receipt, err = bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("Stake tx status is not success")
	}

	return nil
}

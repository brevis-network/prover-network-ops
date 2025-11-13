/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"tools/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type InitializeProverConfig struct {
	SubmitterKeystore   string `mapstructure:"submitter_keystore"`
	SubmitterPassphrase string `mapstructure:"submitter_passphrase"`
	ProverName          string `mapstructure:"prover_name"`
	ProverIcon          string `mapstructure:"prover_icon"`
}

func InitializeProverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init-prover",
		Short: "initialize prover w/o submitter",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initProver()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.MarkFlagRequired(FlagConfig)
	return cmd
}

func init() {
	rootCmd.AddCommand(InitializeProverCmd())
}

func initProver() error {
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

	proverAuth, prover, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "prover CreateTransactOpts")

	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	var s InitializeProverConfig
	err = viper.UnmarshalKey("init_prover", &s)
	chkErr(err, "UnmarshalKey")

	approveAmt := big.NewInt(0)
	var submitterAuth *bind.TransactOpts
	var submitter common.Address
	var brevisMarket *bindings.BrevisMarket
	proverName := strings.TrimSpace(s.ProverName)
	proverIcon := strings.TrimSpace(s.ProverIcon)
	if proverName == "" || proverIcon == "" {
		log.Fatalln("please fill in both prover_name and prover_icon")
	}

	if s.SubmitterKeystore != "" {
		submitterAuth, submitter, err = CreateTransactOpts(s.SubmitterKeystore, s.SubmitterPassphrase, chid)
		chkErr(err, "submitter CreateTransactOpts")
	}
	brevisMarket, err = bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
	chkErr(err, "NewBrevisMarket")
	minSelfStake, err := stakingController.MinSelfStake(nil)
	chkErr(err, "MinSelfStake")
	approveAmt.Add(approveAmt, minSelfStake)

	tx, err := stakingToken.Approve(proverAuth, common.HexToAddress(c.StakingControllerAddr), approveAmt)
	chkErr(err, "Approve")
	log.Printf("approve tx: %s", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("Approve tx status is not success")
	}

	tx, err = stakingController.InitializeProver(proverAuth, 10000) /*defaults to 100% commission for now*/
	checkBrevisCustomError(err, "InitializeProver", bindings.IStakingControllerABI)
	chkErr(err, "InitializeProver")
	log.Printf("InitializeProver tx: %s", tx.Hash())
	receipt, err = bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("InitializeProver tx status is not success")
	}

	tx, err = stakingController.SetProverProfile(proverAuth, proverName, proverIcon)
	checkBrevisCustomError(err, "SetProverProfile", bindings.IStakingControllerABI)
	log.Printf("SetProverProfile tx: %s", tx.Hash())
	receipt, err = bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("SetProverProfile tx status is not success")
	}

	if prover != submitter && submitter != ZeroAddr {
		tx, err := brevisMarket.SetSubmitterConsent(submitterAuth, prover)
		checkBrevisCustomError(err, "SetSubmitterConsent", bindings.IBrevisMarketABI)
		log.Printf("SetSubmitterConsent tx: %s", tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("SetSubmitterConsent tx status is not success")
		}

		tx, err = brevisMarket.RegisterSubmitter(proverAuth, submitter)
		checkBrevisCustomError(err, "RegisterSubmitter", bindings.IBrevisMarketABI)
		log.Printf("RegisterSubmitter tx: %s", tx.Hash())
		receipt, err = bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("RegisterSubmitter tx status is not success")
		}
	}

	return nil
}

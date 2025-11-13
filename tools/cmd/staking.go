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

type StakingConfig struct {
	SubmitterKeystore   string `mapstructure:"submitter_keystore"`
	SubmitterPassphrase string `mapstructure:"submitter_passphrase"`
	ProverName          string `mapstructure:"prover_name"`
	ProverIcon          string `mapstructure:"prover_icon"`

	StakingAmt string `mapstructure:"staking_amt"`
}

const (
	FlagInit = "init"
)

var (
	initialize bool
)

func StakingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake",
		Short: "initialize prover/submitter and stake tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			return stake()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.Flags().BoolVar(&initialize, FlagInit, false, "indicates whether to initialize prover/submitter firstly")
	cmd.MarkFlagRequired(FlagConfig)
	return cmd
}

func init() {
	rootCmd.AddCommand(StakingCmd())
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

	proverAuth, prover, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "prover CreateTransactOpts")

	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	var s StakingConfig
	err = viper.UnmarshalKey("stake", &s)
	chkErr(err, "UnmarshalKey")

	stakingAmt, success := big.NewInt(0).SetString(s.StakingAmt, 0)
	if !success {
		return fmt.Errorf("staking_amt is not a valid number")
	}
	approveAmt := big.NewInt(0).SetBytes(stakingAmt.Bytes())

	var submitterAuth *bind.TransactOpts
	var submitter common.Address
	var brevisMarket *bindings.BrevisMarket
	proverName := strings.TrimSpace(s.ProverName)
	proverIcon := strings.TrimSpace(s.ProverIcon)
	if initialize {
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
	}

	if approveAmt.Sign() == 1 {
		tx, err := stakingToken.Approve(proverAuth, common.HexToAddress(c.StakingControllerAddr), approveAmt)
		chkErr(err, "Approve")
		log.Printf("approve tx: %s", tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("Approve tx status is not success")
		}

		if initialize {
			tx, err := stakingController.InitializeProver(proverAuth, 10000) /*defaults to 100% commission for now*/
			chkErr(err, "InitializeProver")
			log.Printf("InitializeProver tx: %s", tx.Hash())
			receipt, err := bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("InitializeProver tx status is not success")
			}

			tx, err = stakingController.SetProverProfile(proverAuth, proverName, proverIcon)
			chkErr(err, "SetProverProfile")
			log.Printf("SetProverProfile tx: %s", tx.Hash())
			receipt, err = bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("SetProverProfile tx status is not success")
			}
		}

		if stakingAmt.Sign() == 1 {
			tx, err := stakingController.Stake(proverAuth, prover, stakingAmt)
			chkErr(err, "Stake")
			log.Printf("Stake tx: %s", tx.Hash())
			receipt, err := bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("Stake tx status is not success")
			}
		}

		if initialize && prover != submitter && submitter != ZeroAddr {
			tx, err := brevisMarket.SetSubmitterConsent(submitterAuth, prover)
			chkErr(err, "SetSubmitterConsent")
			log.Printf("SetSubmitterConsent tx: %s", tx.Hash())
			receipt, err := bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("SetSubmitterConsent tx status is not success")
			}

			tx, err = brevisMarket.RegisterSubmitter(proverAuth, submitter)
			chkErr(err, "RegisterSubmitter")
			log.Printf("RegisterSubmitter tx: %s", tx.Hash())
			receipt, err = bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("RegisterSubmitter tx status is not success")
			}
		}
	}

	return nil
}

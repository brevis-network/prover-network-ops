/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"tools/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type StakingConfig struct {
	ChainID               uint64 `mapstructure:"chain_id"`
	ChainRpc              string `mapstructure:"chain_rpc"`
	BrevisMarketAddr      string `mapstructure:"brevis_market_addr"`
	StakingTokenAddr      string `mapstructure:"staking_token_addr"`
	StakingControllerAddr string `mapstructure:"staking_controller_addr"`

	ProverKeystore      string `mapstructure:"prover_keystore"`
	ProverPassphrase    string `mapstructure:"prover_passphrase"`
	SubmitterKeystore   string `mapstructure:"submitter_keystore"`
	SubmitterPassphrase string `mapstructure:"submitter_passphrase"`

	StakingAmt        string `mapstructure:"staking_amt"`
	CommissionRateBps uint64 `mapstructure:"commission_rate_bps"`
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

	var c StakingConfig
	err = viper.UnmarshalKey("stake", &c)
	chkErr(err, "UnmarshalKey")

	ec, err := ethclient.Dial(c.ChainRpc)
	chkErr(err, "Dial")
	chid, err := ec.ChainID(context.Background())
	chkErr(err, "ChainID")
	if chid.Uint64() != c.ChainID {
		return fmt.Errorf("chainid mismatch! cfg has %d but onchain has %d", c.ChainID, chid.Uint64())
	}

	proverAuth, prover, err := CreateTransactOpts(c.ProverKeystore, c.ProverPassphrase, chid)
	chkErr(err, "CreateTransactOpts")

	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	stakingAmt, success := big.NewInt(0).SetString(c.StakingAmt, 0)
	if !success {
		return fmt.Errorf("staking_amt is not a valid number")
	}
	approveAmt := big.NewInt(0).SetBytes(stakingAmt.Bytes())

	var submitterAuth *bind.TransactOpts
	var submitter common.Address
	var brevisMarket *bindings.BrevisMarket
	if initialize {
		submitterAuth, submitter, err = CreateTransactOpts(c.SubmitterKeystore, c.SubmitterKeystore, chid)
		chkErr(err, "CreateTransactOpts")
		brevisMarket, err = bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
		chkErr(err, "NewBrevisMarket")
		minSelfStake, err := stakingController.MinSelfStake(nil)
		chkErr(err, "MinSelfStake")
		approveAmt.Add(approveAmt, minSelfStake)
	}

	if approveAmt.Sign() == 1 {
		tx, err := stakingToken.Approve(proverAuth, common.HexToAddress(c.BrevisMarketAddr), approveAmt)
		chkErr(err, "Approve")
		log.Printf("approve tx: %s", tx.Hash())
		receipt, err := bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, "WaitMined")
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalln("Approve tx status is not success")
		}

		if initialize {
			if c.CommissionRateBps > 10000 {
				return fmt.Errorf("commission_rate_bps should not exceed 10000")
			}

			tx, err := stakingController.InitializeProver(proverAuth, c.CommissionRateBps)
			chkErr(err, "InitializeProver")
			log.Printf("InitializeProver tx: %s", tx.Hash())
			receipt, err := bind.WaitMined(context.Background(), ec, tx)
			chkErr(err, "WaitMined")
			if receipt.Status != types.ReceiptStatusSuccessful {
				log.Fatalln("InitializeProver tx status is not success")
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

		if initialize && prover != submitter {
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

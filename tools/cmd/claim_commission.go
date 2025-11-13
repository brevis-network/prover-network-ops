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

func ClaimCommissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-commission",
		Short: "claim accumulated commission (only prover can call)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return claimCommission()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.MarkFlagRequired(FlagConfig)
	return cmd
}

func init() {
	rootCmd.AddCommand(ClaimCommissionCmd())
}

func claimCommission() error {
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

	proverAuth, _, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "prover CreateTransactOpts")

	stakingController, err := bindings.NewIStakingController(common.HexToAddress(c.StakingControllerAddr), ec)
	chkErr(err, "NewIStakingController")

	tx, err := stakingController.ClaimCommission(proverAuth)
	checkBrevisCustomError(err, "ClaimCommission", bindings.IStakingControllerABI)
	log.Printf("ClaimCommission tx: %s", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "WaitMined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("ClaimCommission tx status is not success")
	}

	return nil
}

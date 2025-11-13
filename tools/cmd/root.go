/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

type ChainConfig struct {
	ChainID               uint64 `mapstructure:"chain_id"`
	ChainRpc              string `mapstructure:"chain_rpc"`
	BrevisMarketAddr      string `mapstructure:"brevis_market_addr"`
	StakingTokenAddr      string `mapstructure:"staking_token_addr"`
	StakingControllerAddr string `mapstructure:"staking_controller_addr"`
	MarketViewerAddr      string `mapstructure:"market_viewer_addr"`

	Keystore   string `mapstructure:"keystore"`
	Passphrase string `mapstructure:"passphrase"`
}

const (
	FlagConfig = "config"
)

var (
	config string
)

var rootCmd = &cobra.Command{
	Use:   "tools",
	Short: "",
	Long:  ``,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

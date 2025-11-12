/*
Copyright © 2025 Brevis Network
*/
package cmd

import (
	"context"
	"encoding/json"
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

const (
	FlagReqId = "req-id"
)

var (
	reqId string
)

func RefundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refund",
		Short: "interact with BrevisMarket contract to refund a request that cannot be fulfilled",
		RunE: func(cmd *cobra.Command, args []string) error {
			return refund()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.Flags().StringVar(&reqId, FlagReqId, "", "the req id")
	cmd.MarkFlagRequired(FlagConfig)
	cmd.MarkFlagRequired(FlagReqId)
	return cmd
}

func init() {
	rootCmd.AddCommand(RefundCmd())
}

func refund() error {
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

	auth, _, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "CreateTransactOpts")
	brevisMarket, err := bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
	chkErr(err, "NewBrevisMarket")

	reqIdBytes := common.HexToHash(reqId)
	tx, err := brevisMarket.Refund(auth, reqIdBytes)
	if err != nil {
		var jsonErr JsonError
		errJson, _ := json.Marshal(err)
		json.Unmarshal(errJson, &jsonErr)
		if jsonErr.Data != "" {
			errName, pErr := ParseSolCustomErrorName(bindings.BrevisMarketABI, common.FromHex(jsonErr.Data))
			chkErr(pErr, "ParseSolCustomErrorName")

			log.Fatalf("Refund, err %s - %s", err.Error(), errName)
		} else {
			chkErr(err, "Refund")
		}
	}
	log.Printf("Refund tx: %s", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "waitmined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("Refund tx status is not success")
	}

	return nil
}

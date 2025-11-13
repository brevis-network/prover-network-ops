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

type RefundConfig struct {
	ReqIds []string `mapstructure:"req_ids"`
}

const (
	FlagAll = "all"
)

var (
	all bool
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
	cmd.Flags().BoolVar(&all, FlagAll, false, "indicates whether to refund all refundable requests under my account")
	cmd.MarkFlagRequired(FlagConfig)
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

	auth, sender, err := CreateTransactOpts(c.Keystore, c.Passphrase, chid)
	chkErr(err, "CreateTransactOpts")
	brevisMarket, err := bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
	chkErr(err, "NewBrevisMarket")

	var toRefundReqIds [][32]byte
	if !all {
		var refund RefundConfig
		err = viper.UnmarshalKey("refund", &c)
		chkErr(err, "UnmarshalKey")
		for _, reqId := range refund.ReqIds {
			toRefundReqIds = append(toRefundReqIds, common.HexToHash(reqId))
		}
	} else {
		marketViewer, err := bindings.NewMarketViewer(common.HexToAddress(c.BrevisMarketAddr), ec)
		chkErr(err, "NewMarketViewer")
		toRefundReqIds, err = marketViewer.GetSenderRefundableRequests(nil, sender)
		chkErr(err, "GetSenderRefundableRequests")
	}

	if len(toRefundReqIds) == 0 {
		log.Fatalf("no refundable requests")
	}

	tx, err := brevisMarket.BatchRefund(auth, toRefundReqIds)
	if err != nil {
		var jsonErr JsonError
		errJson, _ := json.Marshal(err)
		json.Unmarshal(errJson, &jsonErr)
		if jsonErr.Data != "" && jsonErr.Data != "0x" {
			errName, pErr := ParseSolCustomErrorName(bindings.BrevisMarketABI, common.FromHex(jsonErr.Data))
			chkErr(pErr, "ParseSolCustomErrorName")

			log.Fatalf("BatchRefund, err %s - %s", err.Error(), errName)
		} else {
			chkErr(err, "BatchRefund")
		}
	}
	log.Printf("BatchRefund tx: %s", tx.Hash())
	receipt, err := bind.WaitMined(context.Background(), ec, tx)
	chkErr(err, "Waitmined")
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalln("BatchRefund tx status is not success")
	}

	return nil
}

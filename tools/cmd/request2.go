package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os/exec"
	"strings"
	"time"
	"tools/bindings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	exeDir, exePath, exeArg string
)

func Request2Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request2",
		Short: "interact with BrevisMarket contract to initiate a proof request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return requestProof2()
		},
	}
	cmd.Flags().StringVar(&config, FlagConfig, "", "config file path")
	cmd.MarkFlagRequired(FlagConfig)
	cmd.Flags().StringVar(&exeDir, "dir", "", "working dir for exec")
	cmd.Flags().StringVar(&exePath, "path", "./target/release/gen-inputs-fibonacci", "relative path to exec")
	cmd.Flags().StringVar(&exeArg, "arg", "--n", "arg")
	return cmd
}

func requestProof2() error {
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
	stakingToken, err := bindings.NewIERC20(common.HexToAddress(c.StakingTokenAddr), ec)
	chkErr(err, "NewIERC20")
	brevisMarket, err := bindings.NewBrevisMarket(common.HexToAddress(c.BrevisMarketAddr), ec)
	chkErr(err, "NewBrevisMarket")

	var reqs Requests
	err = viper.UnmarshalKey("request", &reqs)
	chkErr(err, "UnmarshalKey")
	if len(reqs) == 0 {
		return fmt.Errorf("should provide at least one request")
	}
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

		nonce := uint64(time.Now().Unix()) // same as epoch for uniqueness
		proofReq := getProofReq(nonce)
		proofReq.Fee = bindings.IBrevisMarketFeeParams{
			MaxFee:   feeInt,
			MinStake: minStakeInt,
			Deadline: nonce + 3600,
		}

		tx, err = brevisMarket.RequestProof(auth, proofReq)

		if err != nil {
			var jsonErr JsonError
			errJson, _ := json.Marshal(err)
			json.Unmarshal(errJson, &jsonErr)
			if jsonErr.Data != "" {
				errName, pErr := ParseSolCustomErrorName(bindings.BrevisMarketABI, common.FromHex(jsonErr.Data))
				chkErr(pErr, fmt.Sprintf("req %d: ParseSolCustomErrorName", i+1))

				log.Fatalf("req %d: RequestProof, err %s - %s", i+1, err.Error(), errName)
			} else {
				chkErr(err, fmt.Sprintf("req %d: RequestProof", i+1))
			}
		}
		log.Printf("req %d: RequestProof tx: %s", i+1, tx.Hash())
		receipt, err = bind.WaitMined(context.Background(), ec, tx)
		chkErr(err, fmt.Sprintf("req %d: waitmined", i+1))
		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Fatalf("req %d: RequestProof tx status is not success", i+1)
		}
	}

	return nil
}

/*=== For Onchain ProofRequest ===
vk: 0x00fdadda375301d62070ed545be91cd1317222c10b7b9282197253cce6198da7
publicValuesDigest: 0x1a92390fa6831fa70c4971165f98bf4eb80d48bd499b067b05c07a89d5b91ead
inputData: 0x010000000000000004000000000000000a0000000000000000000000
*/

// set nonce, vk, pv and inputData
func getProofReq(nonce uint64) bindings.IBrevisMarketProofRequest {
	var vk, pv, inputData string
	randArg := fmt.Sprintf("%d", rand.Intn(20)+2)
	cmd := exec.Command(exePath, exeArg, randArg)
	cmd.Dir = exeDir
	output, err := cmd.Output()
	chkErr(err, fmt.Sprintf("run %s/%s %s %s err:", exeDir, exePath, exeArg, randArg))
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "vk:") {
			vk = strings.Split(line, " ")[1]
		}
		if strings.HasPrefix(line, "publicValuesDigest:") {
			pv = strings.Split(line, " ")[1]
		}
		if strings.HasPrefix(line, "inputData:") {
			inputData = strings.Split(line, " ")[1]
		}
	}
	return bindings.IBrevisMarketProofRequest{
		Nonce:              nonce,
		Vk:                 common.HexToHash(vk),
		PublicValuesDigest: common.HexToHash(pv),
		InputData:          common.FromHex(inputData),
	}
}

func init() {
	rootCmd.AddCommand(Request2Cmd())
}

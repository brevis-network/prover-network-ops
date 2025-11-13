package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/celer-network/goutils/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

var ZeroAddr common.Address

func CreateTransactOpts(ksfilePath, passphrase string, chainid *big.Int) (*bind.TransactOpts, common.Address, error) {
	if strings.HasPrefix(ksfilePath, "awskms") {
		kmskeyinfo := strings.SplitN(ksfilePath, ":", 3)
		if len(kmskeyinfo) != 3 {
			return nil, ZeroAddr, fmt.Errorf("%s has wrong format", ksfilePath)
		}
		awskeysec := []string{"", ""}
		if passphrase != "" {
			awskeysec = strings.SplitN(passphrase, ":", 2)
			if len(awskeysec) != 2 {
				return nil, ZeroAddr, fmt.Errorf("%s has wrong format", passphrase)
			}
		}
		kmsSigner, err := eth.NewKmsSigner(kmskeyinfo[1], kmskeyinfo[2], awskeysec[0], awskeysec[1], chainid)
		if err != nil {
			return nil, ZeroAddr, err
		}
		return kmsSigner.NewTransactOpts(), kmsSigner.Addr, nil
	}
	ksBytes, err := os.ReadFile(ksfilePath)
	if err != nil {
		return nil, ZeroAddr, err
	}

	key, err := keystore.DecryptKey(ksBytes, passphrase)
	if err != nil {
		return nil, ZeroAddr, err
	}

	auth, err :=
		bind.NewTransactorWithChainID(strings.NewReader(string(ksBytes)), passphrase, chainid)
	if err != nil {
		return nil, ZeroAddr, err
	}

	return auth, key.Address, err
}

func chkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg+":", err)
	}
}

type JsonError struct {
	Code    int
	Message string
	Data    string
}

func ParseSolCustomErrorName(contractABI string, errData []byte) (string, error) {
	if len(errData) < 4 {
		return "", fmt.Errorf("invalid errData")
	}

	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return "", fmt.Errorf("abi.JSON err: %s", err)
	}

	for _, errDef := range parsedABI.Errors {
		if common.Bytes2Hex(errData[:4]) == common.Bytes2Hex(errDef.ID[:4]) {
			return errDef.Name, nil
		}
	}

	return "", nil
}

func checkBrevisCustomError(err error, logmsg string, contractABI string) {
	if err != nil {
		var jsonErr JsonError
		errJson, _ := json.Marshal(err)
		json.Unmarshal(errJson, &jsonErr)
		if jsonErr.Data != "" && jsonErr.Data != "0x" {
			errName, pErr := ParseSolCustomErrorName(contractABI, common.FromHex(jsonErr.Data))
			chkErr(pErr, "ParseSolCustomErrorName")

			log.Fatalf("%s, err %s - %s", logmsg, err.Error(), errName)
		} else {
			chkErr(err, logmsg)
		}
	}
}

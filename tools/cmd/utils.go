package cmd

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
)

var ZeroAddr common.Address

func CreateTransactOpts(ksfilePath, passphrase string, chainid *big.Int) (*bind.TransactOpts, common.Address, error) {
	ksBytes, err := os.ReadFile(ksfilePath)
	if err != nil {
		return nil, ZeroAddr, err
	}

	key, err := keystore.DecryptKey(ksBytes, passphrase)
	if err != nil {
		return nil, ZeroAddr, err
	}

	submitChainAuth, err :=
		bind.NewTransactorWithChainID(strings.NewReader(string(ksBytes)), passphrase, chainid)
	if err != nil {
		return nil, ZeroAddr, err
	}

	return submitChainAuth, key.Address, err
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

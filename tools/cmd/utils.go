package cmd

import (
	"log"
	"math/big"
	"os"
	"strings"

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

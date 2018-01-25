package main

import (
	"fmt"
	"math/big"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/tests/common"
)

type Browse struct {
}

func (b *Browse) createSetAccountTx1(account string, key *crypto.PrivateKey) *types.Transaction {
	address := accounts.PublicKeyToAddress(*key.Public())
	invokeArgs := []string{
		"SetGlobalState",
		account,
		fmt.Sprintf(`{"add":"%s"}`, address.String()),
	}
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(1), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf("", "invoke", true, []string{}, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}
func (b *Browse) createSetAccountTx2(account string, key *crypto.PrivateKey) *types.Transaction {
	address := accounts.PublicKeyToAddress(*key.Public())
	invokeArgs := []string{
		"SetGlobalState",
		address.String(),
		fmt.Sprintf(`{"acc":"%s"}`, account),
	}
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(1), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf("", "invoke", true, []string{}, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}

func (b *Browse) createAtomicTx(privateKey *crypto.PrivateKey) *types.Transaction {
	txConf := common.NewNormalTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), 1, accounts.PublicKeyToAddress(*privateKey.Public()), privkeyHex)
	tx := common.CreateNormalTransaction(txConf, nil)
	return tx
}

func (b *Browse) init() *types.Transaction {
	assetID := 1
	tx := common.CreateIssueTransaction(issuePriKeyHex, privkeyHex, assetID)
	return tx
}

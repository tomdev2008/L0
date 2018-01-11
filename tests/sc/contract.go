package main

import (
	"fmt"
	"github.com/bocheninc/L0/tests/common"
	"github.com/pborman/uuid"
	"math/big"
	"math/rand"
	"path/filepath"
	"os"
	"github.com/bocheninc/L0/core/types"
	"time"
	//"net/http"
	"strconv"
)

var (
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkeyHex     = "596c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	defaultURL = "http://127.0.0.1:8881"
)

type Contract struct {
	dirPath string
}

func (ct *Contract) createInvokeTransaction() *types.Transaction {
	uid := uuid.NewUUID().String()
	rand.Seed(time.Now().Unix())
	amount := rand.Intn(100)
	return ct.invokeContract("invoke", "./l0coin.lua", []string{}, []string{"send", uid, strconv.Itoa(amount), uid})
}

func (ct *Contract) createInitTransaction() *types.Transaction {
	return  ct.initContract("lua", "./l0coin.lua", []string{"test"}, []string{})
}

func (ct *Contract) invokeContract(typ, contractPath string, initArgs, invokeArgs []string) *types.Transaction {
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), "invoke", false, initArgs, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	//fmt.Println(invokeArgs)
	return tx
}

func (ct *Contract) initContract(typ, contractPath string, initArgs, invokeArgs []string) *types.Transaction {
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), typ, false, initArgs, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	return tx
}

func (ct *Contract) init() *types.Transaction {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Errorf("filePath: %+v", err)
	}
	ct.dirPath = dir //filepath.Join(dir, )


	assetID := 1
	tx := common.CreateIssueTransaction(issuePriKeyHex, privkeyHex, assetID)

	return tx
}


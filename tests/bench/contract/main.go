package main

import (
	"fmt"
	"github.com/bocheninc/L0/tests/common"
	"github.com/pborman/uuid"
	"errors"
	"math/big"
	"math/rand"
	"path/filepath"
	"os"
	"time"
	"strconv"
	"net/http"
)
var (
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	privkeyHex     = "596c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	defaultURL = "http://127.0.0.1:8881"
)

type Contract struct {
	dirPath string
	cnt uint
	hosts []*http.Client
}

func NewContract() *Contract {
	ct := &Contract{
		hosts: make([]*http.Client, 100),
		cnt:0,
	}

	for i:=0; i<2; i++ {
		ct.hosts[i] =  &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1000, //MaxIdleConnections,
			},
			Timeout: time.Duration(200) * time.Second, // RequestTimeout
		}
	}

	return ct
}

func (ct *Contract) transfer() {
	ct.initContract("lua", "./l0coin.lua", []string{"test"}, []string{})
	time.Sleep(3 * time.Second)
	for j := 0; j < 10000; j ++ {
		//time.Sleep(time.Second)
		for i := 0; i<100; i++ {
			//time.Sleep(5 * time.Millisecond)
			uid := uuid.NewUUID().String()
			rand.Seed(time.Now().Unix())
			amount := rand.Intn(100)
			ct.invokeContract("invoke", "./l0coin.lua", []string{}, []string{"send", uid, strconv.Itoa(amount), uid})
		}
	}
	fmt.Println("====finish===")
	os.Exit(0)
}

func (ct *Contract) invokeContract(typ, contractPath string, initArgs, invokeArgs []string) error {
	//ct.cnt++
	//i := ct.cnt%100
	//go func(i int) {
	//	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	//	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), "invoke", false, initArgs, invokeArgs)
	//	if res, err := common.Broadcast(common.CreateContractTransaction(txConf, contractConf), defaultURL,  ct.hosts[i]); err != nil {
	//		fmt.Println(fmt.Sprintf("[Type:%s, Path:%s] CreateContractTransaction Invoke, L0 err: %+v", typ, contractPath, res.Error))
	//	}
	//}(int(i))

	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), "invoke", false, initArgs, invokeArgs)
	tx := common.CreateContractTransaction(txConf, contractConf)
	fmt.Println("tx_hash: ", tx.Hash())
	Relay(NewMsg(0x14, tx.Serialize()))
	return nil
}

func (ct *Contract) initContract(typ, contractPath string, initArgs, invokeArgs []string) error {
	txConf := common.NewContractTxConf([]byte{0}, []byte{0}, big.NewInt(100), big.NewInt(0), privkeyHex)
	contractConf := common.NewContractConf(filepath.Join(ct.dirPath, contractPath), typ, false, initArgs, invokeArgs)
	if res, err := common.Broadcast(common.CreateContractTransaction(txConf, contractConf), defaultURL,  ct.hosts[0]); err != nil {
		return errors.New(fmt.Sprintf("[Type:%s, Path:%s] CreateContractTransaction Init, L0 err: %+v", typ, contractPath, res.Error))
	}

	return nil
}

func (ct *Contract) init() error {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Errorf("filePath: %+v", err)
	}
	ct.dirPath = dir //filepath.Join(dir, )


	assetID := 1
	res, err := common.GetAsset(uint32(assetID))
	if res.Err() != nil {
		if resp, err := common.Broadcast(common.CreateIssueTransaction(issuePriKeyHex, privkeyHex, assetID), defaultURL, ct.hosts[0]); resp.Err() != nil && err != nil {
			return errors.New("issue transactionn fail...")
		}
	}

	return nil

}

func main() {
	srvAddress := []string{
		"127.0.0.1:20166",
		//"127.0.0.1:20167",
		//"127.0.0.1:20168",
		//"127.0.0.1:20169",
		//"127.0.0.1:20170",
	}
	TCPSend(srvAddress)
	ct := NewContract()
	ct.init()
	ct.transfer()
	select {}
}


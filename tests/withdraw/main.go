package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain = []byte{0}
	toChain   = []byte{0}

	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"

	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Second * 500,
	}

	url = "http://localhost:8881"
)

func main() {
	systemPriv, _ := crypto.GenerateKey()
	systemAddr := accounts.PublicKeyToAddress(*systemPriv.Public())
	feePriv, _ := crypto.GenerateKey()
	feeAddr := accounts.PublicKeyToAddress(*feePriv.Public())
	assetID := uint32(time.Now().UnixNano())

	userPriv, _ := crypto.GenerateKey()
	userAddr := accounts.PublicKeyToAddress(*userPriv.Public())

	//模拟交易所提现场景
	//1.发行资产系统账户 10000
	//2.转账给提现账户, 以完成提现操作 5000
	//3.部署提现合约 1000
	//4.发起提现请求
	//5.发起撤销提现请求
	//6.系统账户发起提现成功
	//7.系统账户发起提现失败

	issueTx(systemAddr, assetID, big.NewInt(10000))
	atomicTx(systemPriv, userAddr, assetID, big.NewInt(5000))

	initArgs := []string{}
	initArgs = append(initArgs, systemAddr.String())
	initArgs = append(initArgs, feeAddr.String())
	contractAddr := deployTx(systemPriv, assetID, big.NewInt(0), "./withdraw.lua", initArgs)

	//time.Sleep(10 * time.Second)

	invokeArgs := []string{}
	invokeArgs = append(invokeArgs, "launch")
	invokeArgs = append(invokeArgs, "D0001")
	invokeTx(userPriv, assetID, big.NewInt(1000), contractAddr, invokeArgs)

	invokeArgs = []string{}
	invokeArgs = append(invokeArgs, "cancel")
	invokeArgs = append(invokeArgs, "D0001")
	invokeTx(userPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)

	invokeArgs = []string{}
	invokeArgs = append(invokeArgs, "launch")
	invokeArgs = append(invokeArgs, "D0002")
	invokeTx(userPriv, assetID, big.NewInt(1000), contractAddr, invokeArgs)

	invokeArgs = []string{}
	invokeArgs = append(invokeArgs, "succeed")
	invokeArgs = append(invokeArgs, "D0002")
	invokeArgs = append(invokeArgs, "100")
	invokeTx(systemPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)

	invokeArgs = []string{}
	invokeArgs = append(invokeArgs, "launch")
	invokeArgs = append(invokeArgs, "D0003")
	invokeTx(userPriv, assetID, big.NewInt(1000), contractAddr, invokeArgs)

	invokeArgs = []string{}
	invokeArgs = append(invokeArgs, "fail")
	invokeArgs = append(invokeArgs, "D0003")
	invokeTx(systemPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)
}

func issueTx(owner accounts.Address, assetID uint32, amount *big.Int) {
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	issueSender := accounts.PublicKeyToAddress(*issueKey.Public())
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeIssue,
		uint32(time.Now().UnixNano()),
		issueSender,
		owner,
		assetID,
		amount,
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	issueCoin := make(map[string]interface{})
	issueCoin["id"] = assetID
	tx.Payload, _ = json.Marshal(issueCoin)
	sig, _ := issueKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	fmt.Println("> issuer :", owner.String())
	sendTransaction(tx)
}

func atomicTx(privkey *crypto.PrivateKey, owner accounts.Address, assetID uint32, amount *big.Int) {
	sender := accounts.PublicKeyToAddress(*privkey.Public())
	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeAtomic,
		uint32(time.Now().UnixNano()),
		sender,
		owner,
		assetID,
		amount,
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	fmt.Println("> atomic :", owner.String())
	sendTransaction(tx)
}

func deployTx(privkey *crypto.PrivateKey, assetID uint32, amount *big.Int, path string, args []string) accounts.Address {
	sender := accounts.PublicKeyToAddress(*privkey.Public())

	contractSpec := new(types.ContractSpec)
	f, _ := os.Open(path)
	buf, _ := ioutil.ReadAll(f)
	contractSpec.ContractCode = buf

	var a accounts.Address
	pubBytes := []byte(sender.String() + string(buf))
	a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])
	contractSpec.ContractAddr = a.Bytes()

	contractSpec.ContractParams = args

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeLuaContractInit,
		uint32(time.Now().UnixNano()),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		assetID,
		amount,
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)
	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	fmt.Println("> deploy :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams)
	sendTransaction(tx)

	return a
}

func invokeTx(privkey *crypto.PrivateKey, assetID uint32, amount *big.Int, contractAddr accounts.Address, args []string) {
	sender := accounts.PublicKeyToAddress(*privkey.Public())

	contractSpec := new(types.ContractSpec)
	contractSpec.ContractAddr = contractAddr.Bytes()

	contractSpec.ContractParams = args

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeContractInvoke,
		uint32(time.Now().UnixNano()),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		assetID,
		amount,
		big.NewInt(0),
		uint32(time.Now().Unix()),
	)

	tx.Payload = utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	fmt.Println("> invoke :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams)
	sendTransaction(tx)
}

func httpPost(postForm string, resultHandler func(result map[string]interface{})) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(postForm))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Errorf("Couldn't parse response body. %+v", err))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if resultHandler == nil {
		fmt.Println("http response:", result)
	} else {
		resultHandler(result)
	}
}

func sendTransaction(tx *types.Transaction) {
	fmt.Printf("%s hash: %s, sender: %v, receiver: %v, assetID: %v, amount: %v\n", time.Now().Format("2006-01-02 15:04:05"),
		tx.Hash().String(), tx.Sender(), tx.Recipient(), tx.AssetID(), tx.Amount())
	httpPost(`{"id":1,"method":"Transaction.Broadcast","params":["`+hex.EncodeToString(tx.Serialize())+`"]}`, nil)

}

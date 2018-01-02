package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain      = []byte{0}
	toChain        = []byte{0}
	txChan         = make(chan *types.Transaction, 1)
	issuePriKeyHex = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
)

func main() {
	TCPSend([]string{"127.0.0.1:20166"})
	go func() {
		for {
			select {
			case tx := <-txChan:
				fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "Hash:", tx.Hash(), "Sender:", tx.Sender(), " Nonce: ", tx.Nonce(), "Asset: ", tx.AssetID(), " Type:", tx.GetType(), "txChan size:", len(txChan))
				Relay(NewMsg(0x14, tx.Serialize()))
			}
		}
	}()
	for {
		{
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
		{
			systemPriv, _ := crypto.GenerateKey()
			systemAddr := accounts.PublicKeyToAddress(*systemPriv.Public())
			feePriv, _ := crypto.GenerateKey()
			feeAddr := accounts.PublicKeyToAddress(*feePriv.Public())

			assetID1 := uint32(time.Now().UnixNano())
			userPriv1, _ := crypto.GenerateKey()
			userAddr1 := accounts.PublicKeyToAddress(*userPriv1.Public())

			assetID2 := uint32(time.Now().UnixNano())
			userPriv2, _ := crypto.GenerateKey()
			userAddr2 := accounts.PublicKeyToAddress(*userPriv2.Public())

			//模拟交易所订单清算撮合场景 -- 币币交易
			//1.发行资产1到用户账户 10000
			//2.发行资产2到用户账户 10000
			//3.部署订单清算合约
			//4.用户账户1发起订单请求 1000
			//5.用户账户2发起订单请求 1000
			//6.用户账户2发起撤销订单请求 300

			//5.发起撤销提现请求
			//6.系统账户发起提现成功
			//7.系统账户发起提现失败

			issueTx(userAddr1, assetID1, big.NewInt(10000))
			issueTx(userAddr2, assetID2, big.NewInt(10000))

			initArgs := []string{}
			initArgs = append(initArgs, systemAddr.String())
			initArgs = append(initArgs, feeAddr.String())
			contractAddr := deployTx(systemPriv, uint32(0), big.NewInt(0), "./order.lua", initArgs)

			//time.Sleep(10 * time.Second)

			invokeArgs := []string{}
			invokeArgs = append(invokeArgs, "launch")
			invokeArgs = append(invokeArgs, "D0001")
			invokeTx(userPriv1, assetID1, big.NewInt(1000), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "launch")
			invokeArgs = append(invokeArgs, "D0002")
			invokeTx(userPriv2, assetID2, big.NewInt(1000), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "cancel")
			invokeArgs = append(invokeArgs, "D0002")
			invokeArgs = append(invokeArgs, "500")
			invokeTx(userPriv2, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "matching")
			invokeArgs = append(invokeArgs, "M0001")
			invokeArgs = append(invokeArgs, "D0001")
			invokeArgs = append(invokeArgs, "300")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "matching")
			invokeArgs = append(invokeArgs, "M0001")
			invokeArgs = append(invokeArgs, "D0002")
			invokeArgs = append(invokeArgs, "300")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "matched")
			invokeArgs = append(invokeArgs, "M0001")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "feecharge")
			invokeArgs = append(invokeArgs, "M0001")
			invokeArgs = append(invokeArgs, "D0001")
			invokeArgs = append(invokeArgs, "30")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "syscancel")
			invokeArgs = append(invokeArgs, "D0001")
			invokeArgs = append(invokeArgs, "670")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

			invokeArgs = []string{}
			invokeArgs = append(invokeArgs, "syscancel")
			invokeArgs = append(invokeArgs, "D0002")
			invokeArgs = append(invokeArgs, "200")
			invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)
		}
		time.Sleep(time.Millisecond * 200)
	}

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

	//fmt.Println("> issuer :", owner.String())
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

	//fmt.Println("> atomic :", owner.String())
	sendTransaction(tx)
}

func deployTx(privkey *crypto.PrivateKey, assetID uint32, amount *big.Int, path string, args []string) accounts.Address {
	sender := accounts.PublicKeyToAddress(*privkey.Public())

	contractSpec := new(types.ContractSpec)
	f, _ := os.Open(path)
	buf, _ := ioutil.ReadAll(f)
	contractSpec.ContractCode = buf

	var a accounts.Address
	pubBytes := []byte(time.Now().Format("2006-01-02 15:04:05.999999999") + sender.String() + string(buf))
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
	fmt.Println("> deploy :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams, tx.Hash())
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
	fmt.Println("> invoke :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams, tx.Hash())
	sendTransaction(tx)
}

func sendTransaction(tx *types.Transaction) {
	txChan <- tx
}

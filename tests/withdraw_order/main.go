package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"strconv"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain               = []byte{0}
	toChain                 = []byte{0}
	txChan                  = make(chan *types.Transaction, 1)
	issuePriKeyHex          = "496c663b994c3f6a8e99373c3308ee43031d7ea5120baf044168c95c45fbcf83"
	withdrawSystemPriKeyHex = "e6f6b4ae365c54c93533ae081cb79df81aa674fe02b8142f2bf32c0d02a34781"
	withdrawFeePriKeyHex    = "c924604e4c8e72ac861ab860665cc976e3f0aae29f1d0737c938f853676a5ae2"
	orderSystemPriKeyHex    = "1a115e5b95cd3e21d407ba8799573468ba4248d64e7fec1b412acaf2c5148267"
	orderFeePriKeyHex       = "6586b11f8a6b809f6829dc55a225a7692976cf3eca9d2cbd9efd06ba7de91d4c"
)

func main() {
	atomic := flag.Int("atomic", 0, "atomic tx")
	withdraw := flag.Int("withdraw", 0, "withdraw contract")
	order := flag.Int("order", 0, "order contract")
	flag.Parse()
	if *atomic == 0 && *withdraw == 0 && *order == 0 {
		fmt.Println("Usage: ./withdraw_order -atomic=3 [-withdraw=3] [-order=3]")
		return
	}
	fmt.Println("withdraw_order -atomic=", *atomic, "-withdraw=", *withdraw, "-order=", *order)

	TCPSend([]string{"127.0.0.1:20166"})

	pid := "P" + strconv.FormatInt(int64(os.Getpid()), 10)

	go func() {
		for {
			select {
			case tx := <-txChan:
				fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "Hash:", tx.Hash(), "Sender:", tx.Sender(), " Nonce: ", tx.Nonce(), "Asset: ", tx.AssetID(), " Type:", tx.GetType(), "txChan size:", len(txChan))
				Relay(NewMsg(0x14, tx.Serialize()))
			}
		}
	}()

	if *atomic > 0 {
		//模拟交易所提现场景
		//1.发行资产系统账户 100000000000
		//2.转账 100000000000
		for i := 0; i < *atomic; i++ {
			systemPriv, _ := crypto.GenerateKey()
			systemAddr := accounts.PublicKeyToAddress(*systemPriv.Public())
			assetID := uint32(time.Now().UnixNano())
			issueTx(systemAddr, assetID, big.NewInt(100000000000))
			go func() {
				for {
					userPriv, _ := crypto.GenerateKey()
					userAddr := accounts.PublicKeyToAddress(*userPriv.Public())
					atomicTx(systemPriv, userAddr, assetID, big.NewInt(10))
				}
			}()
		}
	}

	//提现合约合约
	if *withdraw > 0 {
		//模拟交易所提现场景
		//1.部署提现合约

		//2.发行资产系统账户 100000000000
		//3.转账给提现账户, 以完成提现操作 100000000000
		//4.发起提现请求
		//5.发起撤销提现请求
		//6.发起提现请求
		//7.系统账户发起提现成功
		//8.发起提现请求
		//9.系统账户发起提现失败
		systemPriv, _ := crypto.HexToECDSA(withdrawSystemPriKeyHex)
		//systemPriv, _ := crypto.GenerateKey()
		systemAddr := accounts.PublicKeyToAddress(*systemPriv.Public())
		feePriv, _ := crypto.HexToECDSA(withdrawFeePriKeyHex)
		//feePriv, _ := crypto.GenerateKey()
		feeAddr := accounts.PublicKeyToAddress(*feePriv.Public())

		initArgs := []string{}
		initArgs = append(initArgs, systemAddr.String())
		initArgs = append(initArgs, feeAddr.String())
		contractAddr := deployTx(systemPriv, uint32(0), big.NewInt(0), "./withdraw.lua", initArgs)

		//并发执行提现合约
		for i := 0; i < *withdraw; i++ {
			go func(systemPriv *crypto.PrivateKey, index int) {
				assetID := uint32(time.Now().UnixNano())
				userPriv, _ := crypto.GenerateKey()
				userAddr := accounts.PublicKeyToAddress(*userPriv.Public())

				issueTx(systemAddr, assetID, big.NewInt(100000000000))
				atomicTx(systemPriv, userAddr, assetID, big.NewInt(100000000000))

				n := uint64(0)
				for {
					n++
					tid := "I" + strconv.FormatInt(int64(index), 10)
					withdrawID1 := pid + tid + "D" + strconv.FormatUint(n, 10)
					n++
					withdrawID2 := pid + tid + "D" + strconv.FormatUint(n, 10)
					n++
					withdrawID3 := pid + tid + "D" + strconv.FormatUint(n, 10)
					invokeArgs := []string{}
					invokeArgs = append(invokeArgs, "launch")
					invokeArgs = append(invokeArgs, withdrawID1)
					invokeTx(userPriv, assetID, big.NewInt(1000), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "cancel")
					invokeArgs = append(invokeArgs, withdrawID1)
					invokeTx(userPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "launch")
					invokeArgs = append(invokeArgs, withdrawID2)
					invokeTx(userPriv, assetID, big.NewInt(10), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "succeed")
					invokeArgs = append(invokeArgs, withdrawID2)
					invokeArgs = append(invokeArgs, "1")
					invokeTx(systemPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "launch")
					invokeArgs = append(invokeArgs, withdrawID3)
					invokeTx(userPriv, assetID, big.NewInt(1000), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "fail")
					invokeArgs = append(invokeArgs, withdrawID3)
					invokeTx(systemPriv, assetID, big.NewInt(0), contractAddr, invokeArgs)
				}
			}(systemPriv, i)
		}
	}

	//订单清算合约
	if *order > 0 {
		//模拟交易所订单清算撮合场景 -- 币币交易
		//1.部署订单清算合约

		//2.发行资产1到用户账户 100000000000
		//3.发行资产2到用户账户 100000000000
		//4.用户账户1发起订单请求 1000
		//5.用户账户2发起订单请求 1000
		//6.用户账户2发起撤销订单请求 300

		//5.发起撤销提现请求
		//6.系统账户发起提现成功
		//7.系统账户发起提现失败

		systemPriv, _ := crypto.HexToECDSA(orderSystemPriKeyHex)
		//systemPriv, _ := crypto.GenerateKey()
		systemAddr := accounts.PublicKeyToAddress(*systemPriv.Public())
		feePriv, _ := crypto.HexToECDSA(orderFeePriKeyHex)
		//feePriv, _ := crypto.GenerateKey()
		feeAddr := accounts.PublicKeyToAddress(*feePriv.Public())

		initArgs := []string{}
		initArgs = append(initArgs, systemAddr.String())
		initArgs = append(initArgs, feeAddr.String())
		contractAddr := deployTx(systemPriv, uint32(0), big.NewInt(0), "./order.lua", initArgs)

		//并发
		for i := 0; i < *order; i++ {
			go func(systemPriv *crypto.PrivateKey, index int) {
				assetID1 := uint32(time.Now().UnixNano())
				userPriv1, _ := crypto.GenerateKey()
				userAddr1 := accounts.PublicKeyToAddress(*userPriv1.Public())

				assetID2 := uint32(time.Now().UnixNano())
				userPriv2, _ := crypto.GenerateKey()
				userAddr2 := accounts.PublicKeyToAddress(*userPriv2.Public())

				issueTx(userAddr1, assetID1, big.NewInt(100000000000))
				issueTx(userAddr2, assetID2, big.NewInt(100000000000))

				n := uint64(0)
				for {
					tid := "I" + strconv.FormatInt(int64(index), 10)
					n++
					orderID1 := pid + tid + "D" + strconv.FormatUint(n, 10)
					n++
					orderID2 := pid + tid + "D" + strconv.FormatUint(n, 10)
					n++
					matchID1 := pid + tid + "M" + strconv.FormatUint(n, 10)

					invokeArgs := []string{}
					invokeArgs = append(invokeArgs, "launch")
					invokeArgs = append(invokeArgs, orderID1)
					invokeTx(userPriv1, assetID1, big.NewInt(10), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "launch")
					invokeArgs = append(invokeArgs, orderID2)
					invokeTx(userPriv2, assetID2, big.NewInt(20), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "cancel")
					invokeArgs = append(invokeArgs, orderID2)
					invokeArgs = append(invokeArgs, "10")
					invokeTx(userPriv2, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "matching")
					invokeArgs = append(invokeArgs, matchID1)
					invokeArgs = append(invokeArgs, orderID1)
					invokeArgs = append(invokeArgs, "5")
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "matching")
					invokeArgs = append(invokeArgs, matchID1)
					invokeArgs = append(invokeArgs, orderID2)
					invokeArgs = append(invokeArgs, "5")
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "matched")
					invokeArgs = append(invokeArgs, matchID1)
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "feecharge")
					invokeArgs = append(invokeArgs, matchID1)
					invokeArgs = append(invokeArgs, orderID1)
					invokeArgs = append(invokeArgs, "3")
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "syscancel")
					invokeArgs = append(invokeArgs, orderID1)
					invokeArgs = append(invokeArgs, "2")
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)

					invokeArgs = []string{}
					invokeArgs = append(invokeArgs, "syscancel")
					invokeArgs = append(invokeArgs, orderID1)
					invokeArgs = append(invokeArgs, "5")
					invokeTx(systemPriv, uint32(0), big.NewInt(0), contractAddr, invokeArgs)
				}
			}(systemPriv, i)
		}
	}

	c := make(chan struct{})
	<-c
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
	//fmt.Println("> deploy :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams, tx.Hash())
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
	//fmt.Println("> invoke :", accounts.NewAddress(contractSpec.ContractAddr).String(), contractSpec.ContractParams, tx.Hash())
	sendTransaction(tx)
}

func sendTransaction(tx *types.Transaction) {
	txChan <- tx
}

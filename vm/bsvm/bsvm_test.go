package bsvm

import (
	"github.com/bocheninc/L0/nvm"
	"github.com/bocheninc/L0/core/types"
	"math/big"
	"fmt"
	"github.com/pkg/errors"
	"github.com/bocheninc/L0/components/db"
	"io/ioutil"
	"github.com/bocheninc/L0/core/accounts"
	"os"
	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/ledger/state"
	"sync"
	"math/rand"
)

type L0Handler struct {
	sync.Mutex
	cache map[string][]byte
}

func NewL0Handler() *L0Handler {
	return &L0Handler{
		cache: make(map[string][]byte),
	}
}

func (hd *L0Handler)GetGlobalState(key string) ([]byte, error) {
	hd.Lock()
	defer hd.Unlock()

	if value, ok := hd.cache[key]; ok {
		return value, nil
	}
	return []byte{}, errors.New("Not found")
}

func (hd *L0Handler)SetGlobalState(key string, value []byte) error {
	hd.Lock()
	defer hd.Unlock()

	hd.cache[key] = value
	return nil
}

func (hd *L0Handler)DelGlobalState(key string) error {
	hd.Lock()
	defer hd.Unlock()

	delete(hd.cache, key)
	return nil
}

func (hd *L0Handler) ComplexQuery(key string) ([]byte, error) {
	return []byte{}, errors.New("Not found")
}

func (hd *L0Handler)GetState(key string) ([]byte, error) {
	hd.Lock()
	defer hd.Unlock()

	if value, ok := hd.cache[key]; ok {
		return value, nil
	}
	return []byte{}, errors.New("Not found")
}

func (hd *L0Handler) AddState(key string, value []byte) {
	hd.Lock()
	defer hd.Unlock()

	hd.cache[key] = value
	//fmt.Println(hd.cache)
}

func (hd *L0Handler) DelState(key string) {
	hd.Lock()
	defer hd.Unlock()

	delete(hd.cache, key)
}

func (hd *L0Handler)GetByPrefix(prefix string) []*db.KeyValue {
	return []*db.KeyValue{}
}
func (hd *L0Handler)GetByRange(startKey, limitKey string) []*db.KeyValue {
	return []*db.KeyValue{}
}

func (hd *L0Handler) GetBalances(addr string) (*state.Balance, error) {
	hd.Lock()
	defer hd.Unlock()

	balance := state.NewBalance()
	balance.Add(0, big.NewInt(100))
	return balance, nil
}

func (hd *L0Handler) CurrentBlockHeight() uint32 {
	return 100
}

func (hd *L0Handler) AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, txType uint32) {
	hd.Lock()
	defer hd.Unlock()
	fmt.Printf("AddTransfer from:%s to:%s amount:%d txType:%d", fromAddr, toAddr, amount.Int64(), txType)
}

func (hd *L0Handler) SmartContractFailed() {

}

func (hd *L0Handler) SmartContractCommitted() {

}
var fileMap = make(map[string][]byte)
var fileLock sync.Mutex
func CreateContractSpec(args []string, fileName string) *types.ContractSpec {
	contractSpec := new(types.ContractSpec)
	contractSpec.ContractParams = args
	var fileBuf []byte

	fileLock.Lock()
	defer fileLock.Unlock()
	if _, ok := fileMap[fileName]; ok {
		fileBuf = fileMap[fileName]
	} else {
			var err error
			f, _ := os.Open(fileName)
			fileBuf, err = ioutil.ReadAll(f)
			if err != nil {
				fmt.Println("read file failed ....", fileName)
				os.Exit(-1)
			}
			fileMap[fileName] = fileBuf
	}


	//f, _ := os.Open("./l0coin.lua")
	//buf, _ := ioutil.ReadAll(f)

	contractSpec.ContractCode = fileBuf

	var a accounts.Address
	pubBytes := []byte("sender" + string(fileBuf))
	a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])
	contractSpec.ContractAddr = a.Bytes()

	return contractSpec
}

func getRandFile() string {
	num := rand.Intn(2)

	if num % 2 == 0 {
		fmt.Println("0")
		return "l0coin.lua"
	} else {
		fmt.Println("2")
		return "l0coin2.lua"
	}
}

func CreateContractData(args []string) *nvm.ContractData {
	tx := &types.Transaction{}
	tx.Data.Type = types.TypeLuaContractInit
	cs := CreateContractSpec(args, getRandFile())
	return nvm.NewContractData(tx, cs, string(cs.ContractCode))
}

func CreateContractDataWithFileName(args []string, name string, txType uint32) *nvm.ContractData {
	tx := &types.Transaction{}
	tx.Data.Type = txType
	cs := CreateContractSpec(args, name)
	return nvm.NewContractData(tx, cs, string(cs.ContractCode))
}

//
//func TestLuaWorker(t *testing.T) {
//	nvm.VMConf = nvm.DefaultConfig()
//	luaWorker := NewLuaWorker(nvm.DefaultConfig())
//	workerProc := &nvm.WorkerProc{
//		ContractData: CreateContractData([]string{}),
//		PreMethod: "RealInitContract",
//		L0Handler: NewL0Handler(),
//	}
//
//	luaWorker.VmJob(workerProc)
//}
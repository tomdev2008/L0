package jsvm

import (
	"github.com/bocheninc/L0/vm"
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
	//"testing"
	//"github.com/pborman/uuid"
	//"strconv"
	//"math/rand"
	//"time"
	"sync"
	"github.com/bocheninc/L0/components/utils"
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

func (hd *L0Handler)PutGlobalState(key string, value []byte) error {
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

func (hd *L0Handler) GetState(key string) ([]byte, error) {
	hd.Lock()
	defer hd.Unlock()

	if value, ok := hd.cache[key]; ok {
		return value, nil
	}
	return []byte{}, errors.New("Not found")
}

func (hd *L0Handler) PutState(key string, value []byte) error {
	hd.Lock()
	defer hd.Unlock()

	hd.cache[key] = value
	return nil
}

func (hd *L0Handler) DelState(key string) error {
	hd.Lock()
	defer hd.Unlock()

	delete(hd.cache, key)
	return nil
}

func (hd *L0Handler) ComplexQuery(key string) ([]byte, error) {
	return []byte{}, errors.New("Not found")
}

func (hd *L0Handler) GetByPrefix(prefix string) ([]*db.KeyValue, error) {
	return []*db.KeyValue{}, nil
}

func (hd *L0Handler) GetByRange(startKey, limitKey string) ([]*db.KeyValue, error) {
	return []*db.KeyValue{}, nil
}

func (hd *L0Handler) GetBalance(addr string, assetID uint32) (*big.Int, error) {
	return big.NewInt(100), nil
}

func (hd *L0Handler) GetBalances(addr string) (*state.Balance, error) {
	hd.Lock()
	defer hd.Unlock()

	balance := state.NewBalance()
	balance.Amounts[0] = big.NewInt(100)
	balance.Amounts[1] = big.NewInt(50)
	return balance, nil
}

func (hd *L0Handler) GetCurrentBlockHeight() uint32 {
	return 100
}

func (hd *L0Handler) AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, fee *big.Int) error {
	hd.Lock()
	defer hd.Unlock()
	fmt.Printf("AddTransfer from:%s to:%s amount:%d txType:%d", fromAddr, toAddr, amount.Int64(), fee.Int64())
	return nil
}

func (hd *L0Handler) Transfer(tx *types.Transaction) error {
	return nil
}

func (hd *L0Handler) SmartContractFailed() {

}

func (hd *L0Handler) SmartContractCommitted() {

}

func (hd *L0Handler) CombineAndValidRwSet(interface{}) interface{} {
	return nil
}

var fileBuf []byte
func CreateContractSpec(args []string) *types.ContractSpec {
	contractSpec := new(types.ContractSpec)
	contractSpec.ContractParams = args

	if len(fileBuf) == 0 {
		var err error
		f, _ := os.Open("./l0coin.js")
		fileBuf, err = ioutil.ReadAll(f)
		if err != nil {
			fmt.Println("read file failed ....")
			os.Exit(-1)
		}
	}

	//f, _ := os.Open("./l0coin.js")
	//buf, _ := ioutil.ReadAll(f)
	contractSpec.ContractCode = fileBuf

	var a accounts.Address
	pubBytes := []byte("sender" + string(fileBuf))
	a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])
	contractSpec.ContractAddr = a.Bytes()

	return contractSpec
}

func CreateContractData(args []string) *vm.ContractData {
	tx := &types.Transaction{}
	tx.Payload = utils.Serialize(CreateContractSpec(args))
	return vm.NewContractData(tx)
}


//func TestJSWorker(t *testing.T) {
//	nvm.VMConf = nvm.DefaultConfig()
//	jsWorker := NewJsWorker(nvm.DefaultConfig())
//
//	fn := func(data interface{}) interface{} {
//		//fmt.Println(data)
//		return nil
//	}
//
//	initccd := func() *nvm.WorkerProc {
//		uid := uuid.New()
//		amount := strconv.Itoa(rand.Intn(1000))
//		workerProc := &nvm.WorkerProc{
//			ContractData: CreateContractData([]string{uid, amount, uid}),
//			PreMethod: "RealInitContract",
//			L0Handler: NewL0Handler(),
//		}
//
//		return workerProc
//	}
//
//	jsWorker.VmJob(&nvm.WorkerProcWithCallback{
//		WorkProc: initccd(),
//		Fn:fn,
//		})
//
//	time.Sleep(time.Second)
//}
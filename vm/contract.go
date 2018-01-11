package vm

import (
	"math/big"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/core/ledger/state"
)

type ISmartConstract interface {
	GetGlobalState(key string) ([]byte, error)

	PutGlobalState(key string, value []byte) error

	DelGlobalState(key string) error

	GetState(key string) ([]byte, error)

	PutState(key string, value []byte) error

	DelState(key string) error

	GetByPrefix(prefix string) ([]*db.KeyValue, error)

	GetByRange(startKey, limitKey string) ([]*db.KeyValue, error)

	ComplexQuery(key string) ([]byte, error)

	GetBalance(addr string, assetID uint32) (*big.Int, error)

	GetBalances(addr string) (*state.Balance, error)

	GetCurrentBlockHeight() uint32

	AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, fee *big.Int) error

	Transfer(tx *types.Transaction) error

	CallBack(response *state.CallBackResponse) error
}

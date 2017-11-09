package vm

import (
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/core/ledger/state"
	"math/big"
)

type ISmartConstract interface {
	GetGlobalState(key string) ([]byte, error)
	SetGlobalState(key string, value []byte) error
	DelGlobalState(key string) error

	GetState(key string) ([]byte, error)
	AddState(key string, value []byte)
	DelState(key string)
	GetByPrefix(prefix string) []*db.KeyValue
	GetByRange(startKey, limitKey string) []*db.KeyValue
	GetBalances(addr string) (*state.Balance, error)
	CurrentBlockHeight() uint32
	AddTransfer(fromAddr, toAddr string, assetID uint32, amount *big.Int, txType uint32)
	SmartContractFailed()
	SmartContractCommitted()
}

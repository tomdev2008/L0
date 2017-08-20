package msgnet

import (
	"github.com/twinj/uuid"
	"github.com/bocheninc/L0/rpc"
)

const (
	RpcNewAccount = iota + 10
	RpcListAccount
	RpcSignAccount
	RpcTransCreate
	RpcTransBroadcast
	RpcTransBroadQuery
	NetGetLocalpeer
	NetGetPeers
	LedgerBalance
	LedgerHeight
	LastBlockHash
	NumberForBlockHash
	NumberForBlock
	HashForBlock
	HashForTx
	BlockHashForTxs
	BlockNumberForTxs
	MergeTxHashForTxs
)

// Event merge event
type Event interface{}

type requestData struct {
	Data []byte
	IdentityId uuid.Uuid `json:"uuid"`
}

type rpcMessage struct {
	Method string `json:"method"`
}

type transactionCreate struct {
	FromChain string `json:"FromChain"`
	ToChain string `json:"ToChain"`
	Recipient string `json:"Recipient"`
	Nonce uint32 `json:"Nonce"`
	Amount int64 `json:"Amount"`
	Fee    int64 `json:"Fee"`
	TxType uint32 `json:"TxType"`
	PayLoad interface{} `json:"PayLoad"`
}

type PayLoad struct {
	ContractCode string `json:"ContractCode"`
	ContractAddr string `json:"ContractAddr"`
	ContractParams []string `json:"ContractParams"`
}

type accountParams struct {
	AccountType uint32 `json:"AccountType"`
	Passphrase string `json:"Passphrase"`
}


type signParams struct {
	OriginAddr string `json:"OriginTx"`
	Addr string `json:"Addr"`
	PassPhrase string `json:"Pass"`
}

type transactionQuery struct {
	ContractAddr string `json:"ContractAddr"`
	ContractParams []string `json:"ContractParams"`
}

type ledgerTxsByBlockHash struct {
	BlockHash string `json:"BlockHash"`
	TxType uint32 `json:"TxType"`
}

type ledgerTxsByBlockNumber struct {
	BlockNumber uint32 `json:"BlockNumber"`
	TxType uint32 `json:"TxType"`
}

type pmHandler interface {
	SendMsgnetMessage(src, dst string, msg Message) bool
	rpc.INetWorkInfo
	rpc.IBroadcast
	rpc.LedgerInterface
	rpc.AccountInterface
}

// Helper manages merge service
type RpcHelper struct {
	pmSender pmHandler
	rpcAccount *rpc.Account
	rpcTransaction *rpc.Transaction
	rpcNet *rpc.Net
	rpcLedger *rpc.Ledger
}
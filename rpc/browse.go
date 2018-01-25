package rpc

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/log"
	"github.com/hpcloud/tail"
)

//Browse Browse
type Browse struct {
	ledger    LedgerInterface
	txRPC     *Transaction
	netServer INetWorkInfo
	pmHander  IBroadcast
	logList   *list.List
	tempList  *list.List
}

//TheLastBlock for browse
type TheLastBlock struct {
	Hash      string `json:"hash"`
	Height    uint32 `json:"height"`
	MsgnetNum uint32 `json:"msgnetNum"`
	NodeID    string `json:"nodeID"`
	ServerIP  string `json:"serverIP"`
	Status    string `json:"status"`
	TimeStamp uint32 `json:"timeStamp"`
}

//TheRangeBlocks get blocks by range
type TheRangeBlocks struct {
	Blocks []*TheRangeBlock `json:"blocks"`
}
type TheRangeBlock struct {
	Height         uint32 `json:"height"`
	TimeStamp      uint32 `json:"timeStamp"`
	TransactionNum int    `json:"transactionNum"`
}

//BlockInfo get block info by height or hash
type BlockInfo struct {
	Hash           string `json:"hash"`
	Height         uint32 `json:"height"`
	Size           int    `json:"size"`
	TimeStamp      uint32 `json:"timeStamp"`
	TransactionNum int    `json:"transactionNum"`
}
type TxHashList struct {
	TxHashList []*TxHash `json:"txHashList"`
}
type TxHash struct {
	Hash      string `json:"hash"`
	Type      uint32 `json:"type"`
	TimeStamp uint32 `json:"simeStamp"`
}
type GetBlockByHashArgs struct {
	Hash  string `json:"hash"`
	Range []int  `josn:"range"`
}
type GetBlockByHeightArgs struct {
	Height uint32 `json:"height"`
	Range  []int  `josn:"range"`
}

//TxInfo get tx by hash
type TxInfo struct {
	Data     *Data     `json:"data"`
	Contract *Contract `json:"contract"`
}
type Data struct {
	Height      uint32           `json:"height"`
	Amount      *big.Int         `json:"amount"`
	AssetID     uint32           `json:"assetID"`
	Hash        string           `json:"hash"`
	Recipient   accounts.Address `json:"recipient"`
	RecipientID string           `json:"recipientID"`
	Sender      accounts.Address `json:"sender"`
	SenderID    string           `json:"senderID"`
	Size        int              `json:"size"`
	TimeStamp   uint32           `json:"timeStamp"`
	Type        uint32           `json:"type"`
}
type Contract struct {
	Code   []byte           `json:"code"`
	Addr   accounts.Address `json:"addr"`
	Params []string         `json:"params"`
}

//AccountInfo get account info by addr or id
type AccountInfo struct {
	Addr      accounts.Address `json:"address"`
	Balance   *state.Balance   `json:"balance"`
	ID        string           `json:"id"`
	TimeStamp int64            `json:"timeStamp"`
}

type HistoryTxs struct {
	Transactions []*HistoryTx `json:"transactions"`
}
type HistoryTx struct {
	Amount            float64 `json:"amount"`
	AssetID           float64 `json:"assetID"`
	SenderOrRecipient string  `json:"senderOrRecipient"`
	TimeStamp         float64 `json:"timeStamp"`
	Type              float64 `json:"type"`
	Hash              string  `json:"hash"`
}
type GetHistoryTransactionArgs struct {
	Addr  string `json:"addr"`
	Range []int  `json:"range"`
}

//NewBrowse support rpc for Browse browse
func NewBrowse(pm pmHandler) *Browse {
	b := &Browse{
		ledger:    pm,
		txRPC:     NewTransaction(pm),
		netServer: pm,
		pmHander:  pm,
		logList:   list.New(),
		tempList:  list.New(),
	}
	go b.saveLog()
	return b
}

func (l *Browse) saveLog() {
	t, err := tail.TailFile(jrpcCfg.LogFilePath, tail.Config{Follow: true, MustExist: true})
	if err != nil {
		log.Error("browse func saveLog err: ", err)
		return
	}
	n := 0
	for line := range t.Lines {
		l.logList.PushBack(line.Text)
		if l.logList.Len() > jrpcCfg.Logcache {
			e := l.logList.Front()
			if e != nil {
				l.logList.Remove(e)
			}
		}
		n++
	}
}

func (l *Browse) copyLog() {
	if l.tempList.Len() != 0 {
		l.tempList = list.New()
	}
	for e := l.logList.Front(); e != nil; e = e.Next() {
		l.tempList.PushFront(e.Value)
	}
}

func (l *Browse) checkArgs(args []int) (int, int, error) {
	var maxLine = 50
	if len(args) == 1 {
		return 0, 0, errors.New("args len must be 2")
	}
	if len(args) == 2 {
		if args[0] < 0 || (args[1] < 0 && args[1] != -1) {
			return 0, 0, errors.New("params need >= 0, but the second parans can be -1 to query the last ")
		} else if args[1] == -1 || args[1] > maxLine {
			return args[0], maxLine, nil
		} else {
			return args[0], args[1], nil
		}
	}
	return 0, 0, errors.New("params len not bigger two")
}

//GetLog get ldp log for browse
func (l *Browse) GetLog(args []int, reply *[]map[string]interface{}) error {

	result := []map[string]interface{}{}

	start, num, err := l.checkArgs(args)
	if err != nil {
		return err
	}
	func(start, num int) {
		if start == 0 || l.tempList.Len() == 0 {
			l.copyLog()
			valueStr := l.tempList.Front().Value.(string)
			m := make(map[string]interface{})
			err := json.Unmarshal([]byte(valueStr), &m)
			fmt.Println("err: ", err, m)

			result = append(result, m)
		}
		var n int
		for e := l.tempList.Front(); e != nil; e = e.Next() {
			if n > start && n <= (start+num) {
				valueStr := e.Value.(string)
				m := make(map[string]interface{})
				json.Unmarshal([]byte(valueStr), &m)
				result = append(result, m)
			}
			n++
		}
	}(start, num)
	*reply = result
	return nil
}

//GetConfig get ldp config for browse
func (l *Browse) GetConfig(key string, reply *[]byte) error {
	result, err := ioutil.ReadFile(jrpcCfg.ConfigFilePath)
	if err != nil {
		return err
	}
	*reply = result
	return nil
}

//GetTheLastBlockInfo get the last block info  config for browse
func (l *Browse) GetTheLastBlockInfo(ignore string, reply *TheLastBlock) error {
	hash, err := l.ledger.GetLastBlockHash()
	if err != nil {
		return err
	}
	block, err := l.ledger.GetBlockByHash(hash.Bytes())
	if err != nil {
		return err
	}
	*reply = TheLastBlock{
		Hash:      hash.String(),
		Height:    block.Height,
		MsgnetNum: 1,
		NodeID:    string(l.netServer.GetLocalPeer().ID),
		ServerIP:  l.netServer.GetLocalPeer().Address,
		Status:    "运行中",
		TimeStamp: block.TimeStamp,
	}
	return nil
}

//GetBlockByRange get blocks for browse
func (l *Browse) GetBlockByRange(rangeNum int, reply *TheRangeBlocks) error {
	var max = 10
	blocks := &TheRangeBlocks{}
	height, err := l.ledger.Height()
	if err != nil {
		return err

	}
	theLastHeight := int(height)
	f := func(start, num int) error {
		for n := (start + 1); n <= (start + num); n++ {
			block, err := l.ledger.GetBlockByNumber(uint32(n))
			if err != nil {
				return err
			}
			txHashList, err := l.ledger.GetTransactionHashList(block.Height)
			if err != nil {
				return err
			}
			blocks.Blocks = append(blocks.Blocks, &TheRangeBlock{
				Height:         block.Height,
				TimeStamp:      block.TimeStamp,
				TransactionNum: len(txHashList),
			})
		}
		return nil
	}

	if rangeNum == -1 {
		if theLastHeight > max {
			err = f(theLastHeight-max, max)
		} else {
			err = f(0, theLastHeight)
		}
	} else if 0 <= rangeNum && rangeNum <= max {
		if theLastHeight < max {
			err = f(0, theLastHeight)
		} else {
			err = f(0, max)
		}
	} else if rangeNum > max {
		if rangeNum+5 > theLastHeight {
			err = f(theLastHeight-10, max)
		} else {
			err = f(rangeNum-5, max)
		}
	}
	if err != nil {
		return err
	}
	*reply = *blocks
	return nil
}

//GetBlockByHash get block by hash for browse
func (l *Browse) GetBlockByHash(args GetBlockByHashArgs, reply *interface{}) error {
	result, err := l.getBlockByHash(args.Hash, args.Range)
	if err != nil {
		return err
	}
	*reply = result
	return nil
}

//GetBlockByHeight get block by height for browse
func (l *Browse) GetBlockByHeight(args GetBlockByHeightArgs, reply *interface{}) error {
	hash, err := l.ledger.GetBlockHashByNumber(args.Height)
	if err != nil {
		return err
	}
	result, err := l.getBlockByHash(hash.String(), args.Range)
	if err != nil {
		return err
	}
	*reply = result
	return nil
}

func (l *Browse) getBlockByHash(hash string, args []int) (interface{}, error) {
	start, num, err := l.checkArgs(args)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	txHashList := &TxHashList{}
	blockHeader, err := l.ledger.GetBlockByHash(crypto.HexToHash(hash).Bytes())
	if err != nil {
		return nil, err
	}
	txs, err := l.ledger.GetTxsByBlockHash(crypto.HexToHash(hash).Bytes(), 100)
	if err != nil {
		return nil, err
	}
	for n, tx := range txs {
		if n > start && n <= (start+num) {
			txHashList.TxHashList = append(txHashList.TxHashList, &TxHash{
				Hash:      tx.Hash().String(),
				Type:      tx.GetType(),
				TimeStamp: tx.CreateTime(),
			})
		}
	}

	b := &types.Block{Header: blockHeader, Transactions: txs}
	blockInfo := &BlockInfo{
		Hash:           blockHeader.Hash().String(),
		Height:         blockHeader.Height,
		Size:           len(b.Serialize()),
		TimeStamp:      blockHeader.TimeStamp,
		TransactionNum: len(txs),
	}
	m["blocks"] = blockInfo
	m["txHashList"] = txHashList
	return m, nil
}

//GetTxByHash get tx by hash for browse
func (l *Browse) GetTxByHash(hash string, reply *TxInfo) error {
	txInfo := &TxInfo{}
	tx, err := l.ledger.GetTransaction(crypto.HexToHash(hash))
	if err != nil {
		return err
	}
	if tx.GetType() == types.TypeJSContractInit || tx.GetType() == types.TypeLuaContractInit || tx.GetType() == types.TypeContractInvoke {
		contractSpec := &types.ContractSpec{}
		utils.Deserialize(tx.Payload, contractSpec)
		txInfo.Contract = &Contract{
			Addr:   accounts.NewAddress(contractSpec.ContractAddr),
			Code:   contractSpec.ContractCode,
			Params: contractSpec.ContractParams,
		}
	}
	var senderIDStr, recipientIDStr string
	if err := l.txRPC.Query(&ContractQueryArgs{ContractAddr: "", ContractParams: []string{tx.Recipient().String()}}, &recipientIDStr); err != nil {
		return err
	}
	if err := l.txRPC.Query(&ContractQueryArgs{ContractAddr: "", ContractParams: []string{tx.Sender().String()}}, &senderIDStr); err != nil {
		return err
	}
	height, err := l.ledger.GetBlockHeightByTxHash(tx.Hash().Bytes())
	if err != nil {
		return err
	}
	senderID := make(map[string]string)
	err = json.Unmarshal([]byte(senderIDStr), &senderID)
	recipientID := make(map[string]string)
	err = json.Unmarshal([]byte(recipientIDStr), &recipientID)

	fmt.Println("senderID: ", senderIDStr, "err: ", err, "recipientID: ", recipientIDStr, "err: ", err, recipientID["acc"], senderID["acc"])
	txInfo.Data = &Data{
		Height:      height,
		Amount:      tx.Amount(),
		AssetID:     tx.AssetID(),
		Hash:        tx.Hash().String(),
		Recipient:   tx.Recipient(),
		RecipientID: recipientID["acc"],
		Sender:      tx.Sender(),
		SenderID:    senderID["acc"],
		Size:        len(tx.Serialize()),
		TimeStamp:   tx.CreateTime(),
		Type:        tx.GetType(),
	}
	*reply = *txInfo
	return nil
}

//GetAccountInfoByID get Account by id for browse
func (l *Browse) GetAccountInfoByID(accountID string, reply *AccountInfo) error {
	var accountAddrStr string
	err := l.txRPC.Query(&ContractQueryArgs{ContractAddr: "", ContractParams: []string{accountID}}, &accountAddrStr)
	if err != nil {
		return err
	}
	m := make(map[string]string)
	json.Unmarshal([]byte(accountAddrStr), &m)

	balance := l.ledger.GetBalance(accounts.HexToAddress(m["add"]))
	*reply = AccountInfo{
		Addr:      accounts.HexToAddress(m["add"]),
		Balance:   balance,
		ID:        accountID,
		TimeStamp: time.Now().Unix(),
	}
	return nil
}

//GetAccountInfoByAddr get Account by addr for browse
func (l *Browse) GetAccountInfoByAddr(accountAddr string, reply *AccountInfo) error {
	var accountIDstr string
	err := l.txRPC.Query(&ContractQueryArgs{ContractAddr: "", ContractParams: []string{accountAddr}}, &accountIDstr)
	if err != nil {
		return err
	}

	m := make(map[string]string)
	json.Unmarshal([]byte(accountIDstr), &m)
	balance := l.ledger.GetBalance(accounts.HexToAddress(accountAddr))
	*reply = AccountInfo{
		Addr:      accounts.HexToAddress(accountAddr),
		Balance:   balance,
		ID:        m["acc"],
		TimeStamp: time.Now().Unix(), //todo
	}
	return nil
}

//GetHistoryTransaction get history tx by addr for browse
func (l *Browse) GetHistoryTransaction(args GetHistoryTransactionArgs, relay *HistoryTxs) error {
	historyTxs := &HistoryTxs{}
	key := fmt.Sprintf(`db.transaction.find({$or:[{"data.sender":"%s"},{"data.recipient":"%s"}]}).skip(%d).limit(%d).sort({"data.createTime":-1})"]}`, args.Addr, args.Addr, args.Range[0], args.Range[1])
	result, err := l.ledger.ComplexQuery(key)
	if err != nil {
		return err
	}
	maps := []map[string]interface{}{}
	json.Unmarshal(result, &maps)
	for _, v := range maps {
		data := v["data"].(map[string]interface{})
		tx := &HistoryTx{
			AssetID:   data["assetid"].(float64),
			Type:      data["type"].(float64),
			TimeStamp: data["createTime"].(float64),
			Hash:      v["_id"].(string),
		}
		sender := data["sender"].(string)
		recipient := data["recipient"].(string)
		amount := data["amount"].(float64)
		if args.Addr != sender {
			tx.Amount = amount
			tx.SenderOrRecipient = recipient
		} else {
			tx.Amount = -amount
			tx.SenderOrRecipient = sender
		}
		historyTxs.Transactions = append(historyTxs.Transactions, tx)
	}
	*relay = *historyTxs
	return nil
}

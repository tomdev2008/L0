package rpc

import (
	"container/list"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/bocheninc/base/log"
	"github.com/hpcloud/tail"
)

//Browse Browse
type Browse struct {
	ledger    LedgerInterface
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
	TransactionNum uint32 `json:"transactionNum"`
}

//BlockInfo get block info by height or hash
type BlockInfo struct {
	Hash           string    `json:"hash"`
	Height         uint32    `json:"height"`
	Size           int       `json:"size"`
	TimeStamp      int32     `json:"timeStamp"`
	TransactionNum uint32    `json:"transactionNum"`
	TxHashList     []*TxHash `json:"txHashList"`
}

type TyHashList struct {
}
type TxHash struct {
	Hash      string `json:"hash"`
	Type      uint32 `json:"type"`
	TimeStamp uint32 `json:"simeStamp"`
}

//NewBrowse support rpc for Browse browse
func NewBrowse(pm pmHandler) *Browse {
	b := &Browse{
		ledger:    pm,
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
	var n int
	for line := range t.Lines {
		fmt.Println("log ", n, "-->", line.Text)
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
		l.tempList.PushBack(e)
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
func (l *Browse) GetLog(args []int, reply *interface{}) error {
	var (
		result []interface{}
	)
	start, len, err := l.checkArgs(args)
	if err != nil {
		return err
	}
	func(start, len int) {
		if start == 0 {
			l.copyLog()
		}
		var n int
		for e := l.tempList.Front(); e != nil; e = e.Next() {
			if n > start && n < (n+len) {
				result = append(result, e.Value)
			}
			n++
		}
	}(start, len)
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
	theLastHeight, err := l.ledger.Height()
	if err != nil {
		return err
	}

	f := func(start, len uint32) error {
		for start + 1; start <= (start + len); start++ {
			block, err := l.ledger.GetBlockByNumber(start)
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
			err = f(theLastHeight-max, theLastHeight)
		} else {
			err = f(0, theLastHeight)
		}
	} else if rangeNum <= int(theLastHeight) && rangeNum >= 0 {
		if rangeNum < (max / 2) {
			err = f(0, max)
		} else {
			err = f(rangeNum-5, rangeNum+5)
		}
	}
	if err != nil {
		return err
	}
	*reply = *blocks
	return nil
}

//GetBlockByHash get block by hash for browse
func (l *Browse) GetBlockByHash(hash string, reply *interface{}) error {

	return nil
}

//GetBlockByHeight get block by height for browse
func (l *Browse) GetBlockByHeight() error {
	return nil

}

//GetTxByHash get tx by hash for browse
func (l *Browse) GetTxByHash() error {
	return nil

}

//GetAccountInfoByID get Account by id for browse
func (l *Browse) GetAccountInfoByID() error {
	return nil

}

//GetAccountInfoByAddr get Account by addr for browse
func (l *Browse) GetAccountInfoByAddr() error {
	return nil

}

//GetHistoryTransaction get history tx by addr for browse
func (l *Browse) GetHistoryTransaction() error {
	return nil

}

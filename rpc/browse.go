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
	start, end, err := l.checkArgs(args)
	if err != nil {
		return err
	}
	if err = func(line, number int) error {

		return nil
	}(start, end); err != nil {
		return err
	}
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
func (l *Browse) GetTheLastBlockInfo() error {
	return nil
}

//GetBlockByRange get blocks for browse
func (l *Browse) GetBlockByRange() error {
	return nil

}

//GetBlockByHash get block by hash for browse
func (l *Browse) GetBlockByHash() error {
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

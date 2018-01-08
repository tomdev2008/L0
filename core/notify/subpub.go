package notify

import (
	"github.com/bocheninc/L0/core/types"
	"context"
	"sync"
	"time"
	"github.com/bocheninc/L0/components/log"
)

var ps *PubSub
var clientID = "sub-tx"

type tc struct {
	txHash string
	startTime time.Time
}

type PubSub struct {
	txs sync.Map
	ctx context.Context
	txChan chan *tc
}

func init() {
	ps = &PubSub{
		ctx: context.Background(),
		txChan: make(chan *tc, 1000),
	}
	go handleEvent()
}

func RegisterTransaction(tx *types.Transaction) {
	//log.Debugf("Register: %+v", tx.Hash().String())
	ps.txs.Store(tx.Hash().String(), time.Now())
	//ps.s.Subscribe(ps.ctx, clientID, query.MustParse("tm.events.type='NewBlock'"), ps.txChan)
	//ps.s.Subscribe(ps.ctx, clientID, query.Empty{}, ps.txChan)
}

func PublishTransaction(tx *types.Transaction) {
	value, ok := ps.txs.Load(tx.Hash().String())
	//log.Debugf("Publish: %+v, value: %+v, oK: %+v", tx.Hash().String(), value, ok)
	if ok {
		ps.txs.Delete(tx.Hash().String())
		ps.txChan <- &tc{txHash: tx.Hash().String(), startTime: value.(time.Time)}
	}
	//ps.s.PublishWithTags(ps.ctx, clientID, map[string]interface{}{"tm.events.type": "NewBlock"})
	//ps.s.PublishWithTags(ps.ctx, clientID, map[string]interface{}{"tx_hash":tx.Hash().String()})
	//ps.s.Publish(ps.ctx, clientID)
}

var allProcessTransaction time.Duration
var allTransactionCnt int64

func handleEvent() {
	for {
		select {
		case data := <- ps.txChan:
			oneProcessTransaction := time.Now().Sub(data.startTime)
			allProcessTransaction += oneProcessTransaction
			allTransactionCnt += 1
			if allTransactionCnt % 100 == 0 {
				log.Debugf("FinishTransaction, tx_cnt: %+v, avg_time: %d", allTransactionCnt, allProcessTransaction.Nanoseconds() / allTransactionCnt /1000 /1000)

			}
			log.Debugf("FinishTransaction, tx_hash: %+v time: %s", data.txHash, oneProcessTransaction)
		}
	}
}

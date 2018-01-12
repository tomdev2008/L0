package vm

import (
	"sync"
)

var Txsync *TxSync

//type TxSync struct {
//	workerChans map[int]chan struct{}
//	lastIdx int
//	sync.Mutex
//}
//
//func NewTxSync(workersCnt int) *TxSync {
//	txSync := &TxSync{lastIdx: 0}
//	txSync.workerChans = make(map[int]chan struct{})
//	for i := 0; i<workersCnt; i++ {
//		txSync.workerChans[i] = make(chan struct{}, 1)
//	}
//
//	return txSync
//}
//
//
//func (ts *TxSync) Notify(idx int) {
//	//ts.Lock()
//	//defer ts.Unlock()
//
//	ts.workerChans[idx] <- struct{}{}
//}
//
//func (ts *TxSync) Wait(idx int) {
//	//ts.Lock()
//	//defer ts.Unlock()
//
//	<-ts.workerChans[idx]
//}

type TxSync struct {
	workerChans sync.Map
}

func NewTxSync(workersCnt int) *TxSync {
	Txsync = &TxSync{}
	for i := 0; i<workersCnt; i++ {
		Txsync.workerChans.Store(i,  make(chan struct{}, 1))
	}

	return Txsync
}

func (ts *TxSync) Notify(idx int) {
	value, _ := ts.workerChans.Load(idx)
	value.(chan struct{}) <- struct{}{}
}

func (ts *TxSync) Wait(idx int) {
	value, _ := ts.workerChans.Load(idx)
	<-value.(chan struct{})
}
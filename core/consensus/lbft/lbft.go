// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of L0
//
// The L0 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The L0 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package lbft

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/types"
)

//MINQUORUM  Define min quorum
const MINQUORUM = 3

//EMPTYREQUEST empty request id
const EMPTYREQUEST = 1136160000

//NewLbft Create lbft consenter
func NewLbft(options *Options, stack consensus.IStack) *Lbft {
	lbft := &Lbft{
		options: options,
		stack:   stack,

		recvConsensusMsgChan: make(chan *Message, options.BufferSize),
		outputTxsChan:        make(chan *consensus.OutputTxs, options.BufferSize),
		broadcastChan:        make(chan *consensus.BroadcastConsensus, options.BufferSize),
	}
	lbft.height = lbft.stack.GetBlockchainInfo().Height
	lbft.seqNo = lbft.stack.GetBlockchainInfo().LastSeqNo
	lbft.lastExec = lbft.seqNo
	lbft.priority = time.Now().UnixNano()
	lbft.primaryHistory = make(map[string]int64)

	lbft.vcStore = make(map[string]*viewChangeList)
	lbft.coreStore = make(map[string]*lbftCore)
	lbft.committedRequests = make(map[uint32]*Request)

	lbft.blockTimer = time.NewTimer(lbft.options.BlockTimeout)
	lbft.blockTimer.Stop()

	if lbft.options.N < MINQUORUM {
		lbft.options.N = MINQUORUM
	}
	if lbft.options.Request <= lbft.options.ViewChange {
		lbft.options.Request = 3 * lbft.options.ViewChange * 2
	}
	if lbft.options.ViewChangePeriod > 0*time.Second && lbft.options.ViewChangePeriod <= lbft.options.ViewChange {
		lbft.options.ViewChangePeriod = 60 * 3 * lbft.options.ViewChange * 2
	}
	return lbft
}

//Lbft Define lbft consenter
type Lbft struct {
	sync.RWMutex
	options *Options
	stack   consensus.IStack

	recvConsensusMsgChan chan *Message
	outputTxsChan        chan *consensus.OutputTxs
	broadcastChan        chan *consensus.BroadcastConsensus

	height                uint32
	seqNo                 uint32
	lastExec              uint32
	priority              int64
	primaryHistory        map[string]int64
	primaryID             string
	lastPrimaryID         string
	newViewTimer          *time.Timer
	viewChangePeriodTimer *time.Timer

	vcStore           map[string]*viewChangeList
	coreStore         map[string]*lbftCore
	committedRequests map[uint32]*Request

	fetched   []*Committed
	outputTxs types.Transactions
	seqNos    []uint32

	blockTimer *time.Timer
	exit       chan struct{}
}

func (lbft *Lbft) String() string {
	bytes, _ := json.Marshal(lbft.options)
	return string(bytes)
}

//Options
func (lbft *Lbft) Options() consensus.IOptions {
	return lbft.options
}

//Start Start consenter serverice
func (lbft *Lbft) Start() {
	if lbft.exit != nil {
		log.Warnf("Replica %s-%s consenter alreay started", lbft.options.Chain, lbft.options.ID)
		return
	}
	log.Infof("lbft : %s", lbft)
	log.Infof("Replica %s-%s consenter started", lbft.options.Chain, lbft.options.ID)
	lbft.exit = make(chan struct{})
	for {
		select {
		case <-lbft.exit:
			lbft.exit = nil
			log.Infof("Replica %s-%s consenter stopped", lbft.options.Chain, lbft.options.ID)
			return
		case msg := <-lbft.recvConsensusMsgChan:
			for msg != nil {
				msg = lbft.processConsensusMsg(msg)
			}
		case <-lbft.blockTimer.C:
			lbft.height++
			req := &Request{
				ID:     EMPTYREQUEST,
				Time:   uint32(time.Now().Unix()),
				Height: lbft.height,
				Txs:    nil,
				Func:   nil,
			}
			lbft.recvConsensusMsgChan <- &Message{
				Type:    MESSAGEREQUEST,
				Payload: utils.Serialize(req),
			}
		}
	}
}

//Stop Stop consenter serverice
func (lbft *Lbft) Stop() {
	if lbft.exit == nil {
		log.Warnf("Replica %s-%s consenter alreay stopped", lbft.options.Chain, lbft.options.ID)
		return
	}
	close(lbft.exit)
}

// Quorum num of quorum
func (lbft *Lbft) Quorum() int {
	return lbft.options.Q
}

// BatchSize size of batch
func (lbft *Lbft) BatchSize() int {
	return lbft.options.BatchSize
}

// PendingSize size of batch pending
func (lbft *Lbft) PendingSize() int {
	return len(lbft.coreStore)
}

// BatchTimeout size of batch timeout
func (lbft *Lbft) BatchTimeout() time.Duration {
	return lbft.options.BatchTimeout
}

//ProcessBatches
func (lbft *Lbft) ProcessBatch(txs types.Transactions, function func(int, types.Transactions)) {
	lbft.startNewViewTimer()
	if !lbft.isPrimary() {
		function(0, txs)
		return
	}

	lbft.height++
	if success := lbft.stack.VerifyTxs(txs, true); success {
		req := &Request{
			ID:     time.Now().UnixNano(),
			Time:   uint32(time.Now().Unix()),
			Height: lbft.height,
			Txs:    txs,
			Func:   function,
		}
		if !req.isValid() {
			panic("illegal request")
		}
		lbft.recvConsensusMsgChan <- &Message{
			Type:    MESSAGEREQUEST,
			Payload: utils.Serialize(req),
		}
		function(1, txs)
	}
}

//RecvConsensus Receive consensus data for consenter
func (lbft *Lbft) RecvConsensus(payload []byte) {
	msg := &Message{}
	if err := utils.Deserialize(payload, msg); err != nil {
		log.Errorf("Replica %s-%s receive consensus message : unkown %v", lbft.options.Chain, lbft.options.ID, err)
		return
	}
	lbft.recvConsensusMsgChan <- msg
}

//BroadcastConsensusChannel Broadcast consensus data
func (lbft *Lbft) BroadcastConsensusChannel() <-chan *consensus.BroadcastConsensus {
	return lbft.broadcastChan
}

//OutputTxsChannel Commit block data
func (lbft *Lbft) OutputTxsChannel() <-chan *consensus.OutputTxs {
	return lbft.outputTxsChan
}

func (lbft *Lbft) broadcast(to string, msg *Message) {
	lbft.broadcastChan <- &consensus.BroadcastConsensus{
		To:      to,
		Payload: utils.Serialize(msg),
	}
}

func (lbft *Lbft) isPrimary() bool {
	return strings.Compare(lbft.options.ID, lbft.primaryID) == 0
}

func (lbft *Lbft) processConsensusMsg(msg *Message) *Message {
	switch tp := msg.Type; tp {
	case MESSAGEREQUEST:
		if request := msg.GetRequest(); request != nil {
			return lbft.recvRequest(request)
		}
	case MESSAGEPREPREPARE:
		if preprepare := msg.GetPrePrepare(); preprepare != nil {
			return lbft.recvPrePrepare(preprepare)
		}
	case MESSAGEPREPARE:
		if prepare := msg.GetPrepare(); prepare != nil {
			return lbft.recvPrepare(prepare)
		}
	case MESSAGECOMMIT:
		if commit := msg.GetCommit(); commit != nil {
			return lbft.recvCommit(commit)
		}
	case MESSAGECOMMITTED:
		if committed := msg.GetCommitted(); committed != nil {
			return lbft.recvCommitted(committed)
		}
	case MESSAGEFETCHCOMMITTED:
		if fct := msg.GetFetchCommitted(); fct != nil {
			return nil
		}
	case MESSAGEVIEWCHANGE:
		if vc := msg.GetViewChange(); vc != nil {
			return lbft.recvViewChange(vc)
		}
	default:
		log.Warnf("unsupport consensus message type %v ", tp)
	}
	return nil
}

func (lbft *Lbft) startNewViewTimer() {
	lbft.Lock()
	defer lbft.Unlock()
	if lbft.newViewTimer == nil {
		lbft.newViewTimer = time.AfterFunc(lbft.options.Request, func() {
			vc := &ViewChange{
				ID:        "lbft",
				Priority:  lbft.priority,
				PrimaryID: lbft.options.ID,
				SeqNo:     lbft.lastExec,
				Height:    lbft.height,
				OptHash:   lbft.options.Hash(),
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
			}
			lbft.sendViewChange(vc)
		})
	}
}

func (lbft *Lbft) stopNewViewTimer() {
	lbft.Lock()
	defer lbft.Unlock()
	if lbft.newViewTimer != nil {
		lbft.newViewTimer.Stop()
		lbft.newViewTimer = nil
	}
}

func (lbft *Lbft) startViewChangePeriodTimer() {
	if lbft.options.ViewChangePeriod > 0*time.Second && lbft.viewChangePeriodTimer == nil {
		lbft.viewChangePeriodTimer = time.AfterFunc(lbft.options.ViewChangePeriod, func() {
			vc := &ViewChange{
				ID:        "lbft-period",
				Priority:  lbft.priority,
				PrimaryID: lbft.options.ID,
				SeqNo:     lbft.lastExec,
				Height:    lbft.height,
				OptHash:   lbft.options.Hash(),
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
			}
			lbft.sendViewChange(vc)
		})
	}
}

func (lbft *Lbft) stopViewChangePeriodTimer() {
	if lbft.viewChangePeriodTimer != nil {
		lbft.viewChangePeriodTimer.Stop()
		lbft.viewChangePeriodTimer = nil
	}
}

func (lbft *Lbft) recvFetchCommitted(fct *FetchCommitted) *Message {
	if fct.Chain != lbft.options.Chain {
		log.Errorf("Replica %s-%s received FetchCommitted(%d) from %s: ingnore, diff chain (%s-%s)", lbft.options.Chain, lbft.options.ID, fct.SeqNo, fct.ReplicaID, lbft.options.Chain, fct.Chain)
		return nil
	}

	log.Debugf("Replica %s-%s received FetchCommitted(%d) from %s", lbft.options.Chain, lbft.options.ID, fct.SeqNo, fct.ReplicaID)

	if request, ok := lbft.committedRequests[fct.SeqNo]; ok {
		ctt := &Committed{
			Chain:     lbft.options.Chain,
			ReplicaID: lbft.options.ID,
			SeqNo:     fct.SeqNo,
			Request:   request,
		}
		lbft.broadcast(lbft.options.Chain, &Message{
			Type:    MESSAGECOMMITTED,
			Payload: utils.Serialize(ctt),
		})
	} else {
		log.Warnf("Replica %s-%s received FetchCommitted(%d) from %s : ignore missing ", lbft.options.Chain, lbft.options.ID, fct.SeqNo, fct.ReplicaID)
	}
	return nil
}

func (lbft *Lbft) sendViewChange(vc *ViewChange) {
	log.Debugf("Replica %s-%s send ViewChange(%s) for voter %s", lbft.options.Chain, lbft.options.ID, vc.ID, vc.PrimaryID)
	lbft.recvViewChange(vc)
	lbft.broadcast(lbft.options.Chain, &Message{
		Type:    MESSAGEVIEWCHANGE,
		Payload: utils.Serialize(vc),
	})
}

type viewChangeList struct {
	vcs          []*ViewChange
	timeoutTimer *time.Timer
	resendTimer  *time.Timer
}

func (lbft *Lbft) recvViewChange(vc *ViewChange) *Message {
	if vc.Chain != lbft.options.Chain {
		log.Errorf("Replica %s-%s received ViewChange(%s) from %s: ingnore, diff chain (%s-%s)", lbft.options.Chain, lbft.options.ID, vc.ID, vc.ReplicaID, lbft.options.Chain, vc.Chain)
		return nil
	}

	vcl, ok := lbft.vcStore[vc.ID]
	if !ok {
		vcl = &viewChangeList{}
		lbft.vcStore[vc.ID] = vcl
		vcl.timeoutTimer = time.AfterFunc(lbft.options.ViewChange, func() {
			delete(lbft.vcStore, vc.ID)
		})
	} else {
		for _, v := range vcl.vcs {
			if v.Chain == vc.Chain && v.ReplicaID == vc.ReplicaID {
				log.Warningf("Replica %s-%s received received ViewChange(%s) from %s: ingnore, duplicate", lbft.options.Chain, lbft.options.ID, vc.ID, vc.ReplicaID)
				return nil
			}
		}
	}

	log.Debugf("Replica %s-%s received received ViewChange(%s) from %s,  voter: %s %d %d %s", lbft.options.Chain, lbft.options.ID, vc.ID, vc.ReplicaID, vc.PrimaryID, vc.SeqNo, vc.Height, vc.OptHash)

	vcl.vcs = append(vcl.vcs, vc)
	if len(vcl.vcs) >= lbft.Quorum() {
		if len(vcl.vcs) == lbft.Quorum() {
			vcl.timeoutTimer.Stop()
			vcl.timeoutTimer = nil
			vcl.resendTimer = time.AfterFunc(lbft.options.ResendViewChange, func() {
				var tvc *ViewChange
				for _, v := range vcl.vcs {
					if v.PrimaryID == lbft.lastPrimaryID {
						continue
					}
					if v.SeqNo != lbft.lastExec || v.Height != lbft.height || v.OptHash != lbft.options.Hash() {
						continue
					}
					if p, ok := lbft.primaryHistory[tvc.PrimaryID]; ok && p != tvc.Priority {
						continue
					}
					if tvc == nil {
						tvc = v
					} else if tvc.Priority > v.Priority {
						tvc = v
					}
				}
				delete(lbft.vcStore, vc.ID)
				log.Debugf("Replica %s-%s resend ViewChange(%s)", lbft.options.Chain, lbft.options.ID, tvc.ID)
				lbft.sendViewChange(tvc)
			})
			if lbft.primaryID != "" {
				lbft.lastPrimaryID = lbft.primaryID
				lbft.primaryID = ""
			}
		}
		q := 0
		var tvc *ViewChange
		for _, v := range vcl.vcs {
			if v.PrimaryID == lbft.lastPrimaryID {
				continue
			}
			if v.SeqNo != lbft.lastExec || v.Height != lbft.height || v.OptHash != lbft.options.Hash() {
				continue
			}
			if p, ok := lbft.primaryHistory[tvc.PrimaryID]; ok && p != tvc.Priority {
				continue
			}
			if tvc == nil {
				tvc = v
			} else if tvc.Priority > v.Priority {
				tvc = v
			}
			q++
		}
		if q >= lbft.Quorum() {
			vcl.resendTimer.Stop()
			vcl.resendTimer = nil
			lbft.newView(tvc)
		}
	}
	return nil
}

func (lbft *Lbft) newView(vc *ViewChange) {
	lbft.primaryID = vc.PrimaryID
	lbft.seqNo = vc.SeqNo
	lbft.height = vc.Height
	lbft.stopViewChangePeriodTimer()
	lbft.startViewChangePeriodTimer()

	for _, vcl := range lbft.vcStore {
		if vcl.timeoutTimer != nil {
			vcl.timeoutTimer.Stop()
			vcl.timeoutTimer = nil
		}
		if vcl.resendTimer != nil {
			vcl.resendTimer.Stop()
			vcl.resendTimer = nil
		}
	}
	lbft.vcStore = make(map[string]*viewChangeList)

	for _, core := range lbft.coreStore {
		lbft.stopNewViewTimerForCore(core)
		if !core.passCommit && core.prePrepare != nil {
			f := core.prePrepare.Request.Func
			if f != nil {
				f(2, core.prePrepare.Request.Txs)
			}
		}
	}
	lbft.coreStore = make(map[string]*lbftCore)

	for seqNo, req := range lbft.committedRequests {
		if req.Height > lbft.height || seqNo > lbft.seqNo {
			delete(lbft.committedRequests, seqNo)
			f := req.Func
			if f != nil {
				f(2, req.Txs)
			}
		}
	}
}

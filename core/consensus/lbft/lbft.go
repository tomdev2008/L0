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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/log"
)

//MINQUORUM  Define min quorum
const MINQUORUM = 3

//EMPTYREQUEST empty request id
const EMPTYREQUEST = 1136160000

//NewLbft Create lbft consenter
func NewLbft(options *Options, stack consensus.IStack) *Lbft {
	lbft := &Lbft{
		options:    options,
		stack:      stack,
		testing:    true,
		testChan:   make(chan struct{}),
		statistics: make(map[string]time.Duration),

		recvConsensusMsgChan: make(chan *Message, options.BufferSize),
		outputTxsChan:        make(chan *consensus.OutputTxs, options.BufferSize),
		broadcastChan:        make(chan *consensus.BroadcastConsensus, options.BufferSize),
	}
	lbft.primaryHistory = make(map[string]int64)

	lbft.vcStore = make(map[string]*viewChangeList)
	lbft.coreStore = make(map[string]*lbftCore)
	lbft.committedRequests = make(map[uint32]*Committed)

	lbft.blockTimer = time.NewTimer(lbft.options.BlockTimeout)
	lbft.blockTimer.Stop()

	if lbft.options.N < MINQUORUM {
		lbft.options.N = MINQUORUM
	}
	if lbft.options.ResendViewChange < lbft.options.ViewChange {
		lbft.options.ResendViewChange = lbft.options.ViewChange
	}
	if lbft.options.BlockTimeout < lbft.options.BatchTimeout {
		lbft.options.BlockTimeout = lbft.options.BatchTimeout
	}

	if lbft.options.ViewChangePeriod > 0*time.Second && lbft.options.ViewChangePeriod <= lbft.options.ViewChange {
		lbft.options.ViewChangePeriod = 60 * 3 * lbft.options.ViewChange / 2
	}
	return lbft
}

//Lbft Define lbft consenter
type Lbft struct {
	sync.RWMutex
	options       *Options
	stack         consensus.IStack
	testing       bool
	testChan      chan struct{}
	statistics    map[string]time.Duration
	statisticsCnt int

	function func(int, types.Transactions)

	recvConsensusMsgChan chan *Message
	outputTxsChan        chan *consensus.OutputTxs
	broadcastChan        chan *consensus.BroadcastConsensus

	height                uint32
	seqNo                 uint32
	execHeight            uint32
	execSeqNo             uint32
	priority              int64
	primaryHistory        map[string]int64
	primaryID             string
	lastPrimaryID         string
	newViewTimer          *time.Timer
	viewChangePeriodTimer *time.Timer

	rvc               *ViewChange
	vcStore           map[string]*viewChangeList
	rwVcStore         sync.RWMutex
	coreStore         map[string]*lbftCore
	committedRequests map[uint32]*Committed

	fetched []*Committed

	blockTimer *time.Timer
	exit       chan struct{}
}

func (lbft *Lbft) Name() string {
	return "lbft"
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
		log.Warnf("Replica %s consenter already started", lbft.options.ID)
		return
	}
	if lbft.testing {
		lbft.testConsensus()
	}
	log.Infof("lbft : %s", lbft)
	log.Infof("Replica %s consenter started", lbft.options.ID)
	lbft.height = lbft.stack.GetBlockchainInfo().Height
	lbft.seqNo = lbft.stack.GetBlockchainInfo().LastSeqNo
	lbft.execHeight = lbft.height
	lbft.execSeqNo = lbft.seqNo
	lbft.priority = time.Now().UnixNano()
	lbft.exit = make(chan struct{})
	for {
		select {
		case <-lbft.exit:
			lbft.exit = nil
			log.Infof("Replica %s consenter stopped", lbft.options.ID)
			return
		case msg := <-lbft.recvConsensusMsgChan:
			for msg != nil {
				msg = lbft.processConsensusMsg(msg)
			}
		case <-lbft.blockTimer.C:
			lbft.sendEmptyRequest()
		}
	}
}

func (lbft *Lbft) sendEmptyRequest() {
	if lbft.isPrimary() {
		lbft.blockTimer.Stop()
		req := &Request{
			ID:     EMPTYREQUEST,
			Time:   uint32(time.Now().UnixNano()),
			Height: lbft.height,
			Txs:    nil,
		}
		// lbft.recvConsensusMsgChan <- &Message{
		// 	Type:    MESSAGEREQUEST,
		// 	Payload: utils.Serialize(req),
		// }
		lbft.recvRequest(req)
	}
}

//Stop Stop consenter serverice
func (lbft *Lbft) Stop() {
	if lbft.exit == nil {
		log.Warnf("Replica %s consenter already stopped", lbft.options.ID)
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
	lbft.function = function
	if len(txs) == 0 {
		return
	}
	lbft.startNewViewTimer()
	if !lbft.isPrimary() {
		lbft.function(0, txs)
		return
	}
	lbft.function(1, txs)
	lbft.function(3, txs)
	req := &Request{
		ID:   time.Now().UnixNano(),
		Time: uint32(time.Now().Unix()),
		Txs:  txs,
	}
	log.Debugf("Replica %s send Request for consensus %s", lbft.options.ID, req.Name())
	lbft.recvConsensusMsgChan <- &Message{
		Type:    MESSAGEREQUEST,
		Payload: utils.Serialize(req),
	}
}

//RecvConsensus Receive consensus data for consenter
func (lbft *Lbft) RecvConsensus(payload []byte) {
	msg := &Message{}
	if err := utils.Deserialize(payload, msg); err != nil {
		log.Errorf("Replica %s receive consensus message : unkown %v", lbft.options.ID, err)
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
	if lbft.testing {
		return nil
	}
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

func (lbft *Lbft) hasPrimary() bool {
	return strings.Compare("", lbft.primaryID) != 0
}

func (lbft *Lbft) processConsensusMsg(msg *Message) *Message {
	log.Debugf("lbft handle consensus message type %v ", msg.Type)
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
		id := time.Now().Truncate(lbft.options.Request).Format("2006-01-02 15:04:05")
		lbft.newViewTimer = time.AfterFunc(lbft.options.Request, func() {
			lbft.Lock()
			defer lbft.Unlock()
			vc := &ViewChange{
				ID:            "lbft-" + id,
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.primaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("request timeout(%s)", lbft.options.Request))
			lbft.newViewTimer = nil
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
				ID:            "lbft-period",
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.primaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("period timemout(%v)", lbft.options.ViewChangePeriod))
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
		log.Errorf("Replica %s received FetchCommitted(%d) from %s: ingnore, diff chain (%s-%s)", lbft.options.ID, fct.SeqNo, fct.ReplicaID, lbft.options.Chain, fct.Chain)
		return nil
	}

	log.Debugf("Replica %s received FetchCommitted(%d) from %s", lbft.options.ID, fct.SeqNo, fct.ReplicaID)

	if request, ok := lbft.committedRequests[fct.SeqNo]; ok {
		ctt := &Committed{
			SeqNo:     fct.SeqNo,
			Height:    request.Height,
			Digest:    request.Digest,
			Txs:       request.Txs,
			ErrTxs:    request.ErrTxs,
			Chain:     lbft.options.Chain,
			ReplicaID: lbft.options.ID,
		}
		lbft.broadcast(lbft.options.Chain, &Message{
			Type:    MESSAGECOMMITTED,
			Payload: utils.Serialize(ctt),
		})
	} else {
		log.Warnf("Replica %s received FetchCommitted(%d) from %s : ignore missing ", lbft.options.ID, fct.SeqNo, fct.ReplicaID)
	}
	return nil
}

func (lbft *Lbft) sendViewChange(vc *ViewChange, reason string) {
	log.Infof("Replica %s send ViewChange(%s) for voter %s: %s", lbft.options.ID, vc.ID, vc.PrimaryID, reason)
	msg := &Message{
		Type:    MESSAGEVIEWCHANGE,
		Payload: utils.Serialize(vc),
	}
	lbft.recvConsensusMsgChan <- msg
	//lbft.recvViewChange(vc)
	lbft.broadcast(lbft.options.Chain, msg)
}

type viewChangeList struct {
	vcs          []*ViewChange
	timeoutTimer *time.Timer
	resendTimer  *time.Timer
}

func (vcl *viewChangeList) start(lbft *Lbft) {
	vcl.timeoutTimer = time.AfterFunc(lbft.options.ViewChange, func() {
		lbft.rwVcStore.Lock()
		vcs := vcl.vcs
		delete(lbft.vcStore, vcs[0].ID)
		lbft.rwVcStore.Unlock()
		if len(vcs) >= lbft.Quorum() {
			var tvc *ViewChange
			for _, v := range vcs {
				if v.PrimaryID == lbft.lastPrimaryID {
					continue
				}
				if (lbft.execSeqNo != 0 && v.SeqNo <= lbft.execSeqNo) || v.Height < lbft.execHeight || v.OptHash != lbft.options.Hash() {
					continue
				}
				if p, ok := lbft.primaryHistory[v.PrimaryID]; ok && p != v.Priority {
					continue
				}
				if tvc == nil {
					tvc = v
				} else if tvc.Priority > v.Priority {
					tvc = v
				}
			}
			log.Infof("Replica %s ViewChange(%s) timeout %s : voter %v", lbft.options.ID, vcs[0].ID, lbft.options.ViewChange, tvc)
			if tvc != nil && lbft.rvc == nil {
				lbft.rvc = tvc
				//vcl.resendTimer = time.AfterFunc(lbft.options.ResendViewChange, func() {
				lbft.rvc.ID += ":resend-" + tvc.PrimaryID
				lbft.rvc.Chain = lbft.options.Chain
				lbft.rvc.ReplicaID = lbft.options.ID
				lbft.sendViewChange(lbft.rvc, fmt.Sprintf("resend timeout(%s) - %s", lbft.options.ResendViewChange, tvc.ID))
				lbft.rvc = nil
				//})
			}
		} else {
			log.Debugf("Replica %s ViewChange(%s) timeout %s : %d", lbft.options.ID, vcs[0].ID, lbft.options.ViewChange, len(vcs))
		}
	})
}

func (vcl *viewChangeList) stop() {
	if vcl.timeoutTimer != nil {
		vcl.timeoutTimer.Stop()
		vcl.timeoutTimer = nil
	}
	if vcl.resendTimer != nil {
		vcl.resendTimer.Stop()
		vcl.resendTimer = nil
	}
}

func (lbft *Lbft) recvViewChange(vc *ViewChange) *Message {
	if vc.Chain != lbft.options.Chain {
		log.Errorf("Replica %s received ViewChange(%s) from %s: ingnore, diff chain (%s-%s)", lbft.options.ID, vc.ID, vc.ReplicaID, lbft.options.Chain, vc.Chain)
		return nil
	}

	// if len(lbft.primaryID) != 0 && vc.LastPrimaryID != lbft.primaryID {
	// 	log.Errorf("Replica %s received ViewChange(%s) from %s: ingnore, diff primaryID (%s-%s)", lbft.options.ID, vc.ID, vc.ReplicaID, lbft.primaryID, vc.LastPrimaryID)
	// 	return nil
	// }

	lbft.rwVcStore.Lock()
	defer lbft.rwVcStore.Unlock()
	vcl, ok := lbft.vcStore[vc.ID]
	if !ok {
		vcl = &viewChangeList{}
		lbft.vcStore[vc.ID] = vcl
		vcl.start(lbft)
	} else {
		for _, v := range vcl.vcs {
			if v.Chain == vc.Chain && v.ReplicaID == vc.ReplicaID {
				log.Warningf("Replica %s received ViewChange(%s) from %s: ingnore, duplicate, size %d", lbft.options.ID, vc.ID, vc.ReplicaID, len(vcl.vcs))
				//lbft.rwVcStore.Unlock()
				return nil
			}
		}
	}
	vcl.vcs = append(vcl.vcs, vc)
	vcs := vcl.vcs
	//lbft.rwVcStore.Unlock()

	// if _, ok := lbft.primaryHistory[vc.PrimaryID]; !ok && vc.PrimaryID == vc.ReplicaID {
	// 	lbft.primaryHistory[vc.PrimaryID] = vc.Priority
	// }
	log.Infof("Replica %s received ViewChange(%s) from %s,  voter: %s %d %d %s, self: %d %d %s, size %d", lbft.options.ID, vc.ID, vc.ReplicaID, vc.PrimaryID, vc.SeqNo, vc.Height, vc.OptHash, lbft.execSeqNo, lbft.execHeight, lbft.options.Hash(), len(vcs))

	if len(vcs) >= lbft.Quorum() {
		lbft.stopNewViewTimer()
		// if len(vcs) == lbft.Quorum() {
		// 	if lbft.primaryID != "" {
		// 		lbft.lastPrimaryID = lbft.primaryID
		// 		lbft.primaryID = ""
		// 		log.Infof("Replica %s ViewChange(%s) over : clear PrimaryID %s - %s", lbft.options.ID, vcs[0].ID, lbft.lastPrimaryID, vcs[0].ID)
		// 	}
		// }
		q := 0
		var tvc *ViewChange
		for _, v := range vcs {
			if v.PrimaryID == lbft.lastPrimaryID {
				continue
			}
			if (lbft.execSeqNo != 0 && v.SeqNo <= lbft.execSeqNo) || v.Height < lbft.execHeight || v.OptHash != lbft.options.Hash() {
				continue
			}
			if p, ok := lbft.primaryHistory[v.PrimaryID]; ok && p != v.Priority {
				continue
			}
			if tvc == nil {
				tvc = v
			} else if tvc.Priority > v.Priority {
				tvc = v
			}
		}
		for _, v := range vcs {
			if v.PrimaryID == lbft.lastPrimaryID {
				continue
			}
			if (lbft.execSeqNo != 0 && v.SeqNo <= lbft.execSeqNo) || v.Height < lbft.execHeight || v.OptHash != lbft.options.Hash() {
				continue
			}
			if p, ok := lbft.primaryHistory[v.PrimaryID]; ok && p != v.Priority {
				continue
			}
			if v.PrimaryID != tvc.PrimaryID {
				continue
			}
			q++
		}
		if q >= lbft.Quorum() && lbft.primaryID == "" {
			if lbft.primaryID != "" {
				lbft.lastPrimaryID = lbft.primaryID
				lbft.primaryID = ""
				log.Infof("Replica %s ViewChange(%s) over : clear PrimaryID %s - %s", lbft.options.ID, vcs[0].ID, lbft.lastPrimaryID, vcs[0].ID)
			}
			lbft.newView(tvc)
		}
	}
	return nil
}

func (lbft *Lbft) newView(vc *ViewChange) {
	log.Infof("Replica %s vote new PrimaryID %s (%d %d) --- %s", lbft.options.ID, vc.PrimaryID, vc.SeqNo, vc.Height, vc.ID)
	lbft.primaryID = vc.PrimaryID
	lbft.seqNo = vc.SeqNo
	lbft.height = vc.Height
	lbft.execSeqNo = lbft.seqNo
	lbft.execHeight = lbft.height
	delete(lbft.primaryHistory, lbft.primaryID)
	if lbft.primaryID == lbft.options.ID {
		lbft.priority = time.Now().UnixNano()
	}
	lbft.stopViewChangePeriodTimer()
	lbft.startViewChangePeriodTimer()

	for _, vcl := range lbft.vcStore {
		vcl.stop()
	}
	lbft.vcStore = make(map[string]*viewChangeList)
	for _, core := range lbft.coreStore {
		lbft.stopNewViewTimerForCore(core)
		if core.prePrepare != nil {
			lbft.function(5, core.txs)
		}
	}
	lbft.coreStore = make(map[string]*lbftCore)

	for seqNo, req := range lbft.committedRequests {
		if req.Height > lbft.execHeight || seqNo > lbft.execSeqNo {
			delete(lbft.committedRequests, seqNo)
			lbft.function(5, req.Txs)
		}
	}
}

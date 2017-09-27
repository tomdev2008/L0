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
	"fmt"
	"sort"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/types"
)

type lbftCore struct {
	digest       string
	fromChain    string
	toChain      string
	prePrepare   *PrePrepare
	prepare      []*Prepare
	passPrepare  bool
	commit       []*Commit
	passCommit   bool
	newViewTimer *time.Timer
}

func (lbft *Lbft) getlbftCore(digest string) *lbftCore {
	core, ok := lbft.coreStore[digest]
	if ok {
		return core
	}

	core = &lbftCore{
		digest: digest,
	}
	lbft.coreStore[digest] = core
	return core
}

func (lbft *Lbft) startNewViewTimerForCore(core *lbftCore) {
	lbft.stopNewViewTimer()
	if core.newViewTimer == nil {
		core.newViewTimer = time.AfterFunc(lbft.options.Request, func() {
			vc := &ViewChange{
				ID:        core.digest,
				Priority:  lbft.priority,
				PrimaryID: lbft.options.ID,
				SeqNo:     lbft.execSeqNo,
				Height:    lbft.execHeight,
				OptHash:   lbft.options.Hash(),
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
			}
			lbft.sendViewChange(vc, "timeout")
		})
	}
}

func (lbft *Lbft) stopNewViewTimerForCore(core *lbftCore) {
	if core.newViewTimer != nil {
		core.newViewTimer.Stop()
		core.newViewTimer = nil
	}
}

func (lbft *Lbft) maybePassPrepare(core *lbftCore) bool {
	nfq := 0
	ntq := 0
	fq := 0
	tq := 0
	self := false
	hasPrimary := false
	for _, prepare := range core.prepare {
		if prepare.Chain != core.fromChain && prepare.Chain != core.toChain {
			continue
		}
		if lbft.options.Chain == prepare.Chain {
			if core.prePrepare.SeqNo != prepare.SeqNo || core.prePrepare.PrimaryID != prepare.PrimaryID ||
				core.prePrepare.Height != prepare.Height || core.prePrepare.OptHash != prepare.OptHash {
				continue
			}
			if prepare.ReplicaID == lbft.options.ID {
				self = true
			}
			if prepare.ReplicaID == prepare.PrimaryID {
				hasPrimary = true
			}
		}
		if prepare.Chain == core.fromChain {
			nfq++
			fq = prepare.Quorum
		} else if prepare.Chain == core.toChain {
			ntq++
			tq = prepare.Quorum
		}
	}
	log.Debugf("Replica %s received Prepare for consensus %s, voted: %d(%d/%d,%d/%d,%v)", lbft.options.ID, core.digest, len(core.prepare), fq, nfq, tq, ntq, self)
	return hasPrimary && self && nfq >= fq && ntq >= tq
}

func (lbft *Lbft) maybePassCommit(core *lbftCore) bool {
	nfq := 0
	ntq := 0
	fq := 0
	tq := 0
	self := false
	hasPrimary := false
	for _, commit := range core.commit {
		if commit.Chain != core.fromChain && commit.Chain != core.toChain {
			continue
		}
		if lbft.options.Chain == commit.Chain {
			if core.prePrepare.SeqNo != commit.SeqNo || core.prePrepare.PrimaryID != commit.PrimaryID ||
				core.prePrepare.Height != commit.Height || core.prePrepare.OptHash != commit.OptHash {
				continue
			}
			if commit.ReplicaID == lbft.options.ID {
				self = true
			}
			if commit.ReplicaID == commit.PrimaryID {
				hasPrimary = true
			}
		}
		if commit.Chain == core.fromChain {
			nfq++
			fq = commit.Quorum
		} else if commit.Chain == core.toChain {
			ntq++
			tq = commit.Quorum
		}
	}
	log.Debugf("Replica %s received Commit for consensus %s, voted: %d(%d/%d,%d/%d,%v)", lbft.options.ID, core.digest, len(core.commit), fq, nfq, tq, ntq, self)
	return hasPrimary && self && nfq >= fq && ntq >= tq
}

func (lbft *Lbft) recvRequest(request *Request) *Message {
	digest := request.Name()
	if lbft.isPrimary() {
		fromChain := request.fromChain()
		toChain := request.toChain()
		if request.ID == EMPTYREQUEST {
			fromChain = lbft.options.Chain
			toChain = lbft.options.Chain
		}
		if !request.isValid() || (fromChain != lbft.options.Chain && toChain != lbft.options.Chain) {
			log.Errorf("Replica %s received Request for consensus %s: ignore, illegal request", lbft.options.ID, digest)
			return nil
		}

		if lbft.options.Chain != toChain {
			lbft.broadcast(toChain, &Message{
				Type:    MESSAGEREQUEST,
				Payload: utils.Serialize(request),
			})
		} else if fromChain != toChain {
			if !lbft.stack.VerifyTxs(request.Txs, true) {
				log.Errorf("Replica %s received Request for consensus %s: ignore, failed to verify", lbft.options.ID, digest)
				return nil
			}
		}

		seqNo := lbft.seqNo + 1
		height := lbft.height
		if lbft.cnt == 0 {
			height = lbft.height + 1
		}
		lbft.cnt += len(request.Txs)
		if lbft.cnt >= lbft.options.BlockSize || request.ID == EMPTYREQUEST {
			lbft.cnt = 0
		}

		log.Debugf("Replica %s received Request for consensus %s", lbft.options.ID, digest)

		preprepare := &PrePrepare{
			PrimaryID: lbft.primaryID,
			SeqNo:     seqNo,
			Height:    height,
			OptHash:   lbft.options.Hash(),
			//Digest:    digest,
			Quorum:    lbft.Quorum(),
			Request:   request,
			Chain:     lbft.options.Chain,
			ReplicaID: lbft.options.ID,
		}

		log.Debugf("Replica %s send PrePrepare for consensus %s", lbft.options.ID, digest)
		lbft.recvPrePrepare(preprepare)
		lbft.broadcast(lbft.options.Chain, &Message{
			Type:    MESSAGEPREPREPARE,
			Payload: utils.Serialize(preprepare),
		})
	} else {
		log.Debugf("Replica %s received Request for consensus %s: ignore, backup", lbft.options.ID, digest)
	}
	return nil
}

func (lbft *Lbft) recvPrePrepare(preprepare *PrePrepare) *Message {
	if preprepare.Request == nil {
		return nil
	}
	digest := preprepare.Request.Name()
	if preprepare.Chain != lbft.options.Chain {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, diff chain (%s==%s)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.Chain, lbft.options.Chain)
		return nil
	}
	if preprepare.ReplicaID != lbft.primaryID {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, diff primayID (%s==%s)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.PrimaryID, lbft.primaryID)
		return nil
	}

	if preprepare.SeqNo != lbft.seqNo+1 {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, wrong seqNo (%d==%d)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.SeqNo, lbft.seqNo+1)
		vc := &ViewChange{
			ID:        digest,
			Priority:  lbft.priority,
			PrimaryID: lbft.options.ID,
			SeqNo:     lbft.execSeqNo,
			Height:    lbft.execHeight,
			OptHash:   lbft.options.Hash(),
			ReplicaID: lbft.options.ID,
			Chain:     lbft.options.Chain,
		}
		lbft.sendViewChange(vc, fmt.Sprintf("wrong seqNo (%d==%d)", preprepare.SeqNo, lbft.seqNo+1))
		return nil
	}

	core := lbft.getlbftCore(digest)
	if core.prePrepare != nil {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: alreay exist ", lbft.options.ID, preprepare.ReplicaID, digest)
		vc := &ViewChange{
			ID:        digest,
			Priority:  lbft.priority,
			PrimaryID: lbft.options.ID,
			SeqNo:     lbft.execSeqNo,
			Height:    lbft.execHeight,
			OptHash:   lbft.options.Hash(),
			ReplicaID: lbft.options.ID,
			Chain:     lbft.options.Chain,
		}
		lbft.sendViewChange(vc, fmt.Sprintf("alreay exist"))
		return nil
	}

	fromChain := preprepare.Request.fromChain()
	toChain := preprepare.Request.toChain()
	if preprepare.Request.ID == EMPTYREQUEST {
		fromChain = lbft.options.Chain
		toChain = lbft.options.Chain
	}
	if !lbft.isPrimary() {
		if !preprepare.Request.isValid() || (fromChain != lbft.options.Chain && toChain != lbft.options.Chain) {
			log.Errorf("Replica %s received PrePrepare from %s for consensus %s: illegal request", lbft.options.ID, preprepare.ReplicaID, digest)
			vc := &ViewChange{
				ID:        digest,
				Priority:  lbft.priority,
				PrimaryID: lbft.options.ID,
				SeqNo:     lbft.execSeqNo,
				Height:    lbft.execHeight,
				OptHash:   lbft.options.Hash(),
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("illegal request"))
			return nil
		}

		if !lbft.stack.VerifyTxs(preprepare.Request.Txs, false) {
			log.Errorf("Replica %s received PrePrepare from %s for consensus %s: failed to verify", lbft.options.ID, preprepare.ReplicaID, digest)
			vc := &ViewChange{
				ID:        digest,
				Priority:  lbft.priority,
				PrimaryID: lbft.options.ID,
				SeqNo:     lbft.execSeqNo,
				Height:    lbft.execHeight,
				OptHash:   lbft.options.Hash(),
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("failed to verify"))
			return nil
		}
	}

	log.Debugf("Replica %s received PrePrepare from %s for consensus %s", lbft.options.ID, preprepare.ReplicaID, digest)

	lbft.startNewViewTimerForCore(core)
	core.fromChain = fromChain
	core.toChain = toChain
	core.prePrepare = preprepare
	lbft.seqNo = preprepare.SeqNo
	lbft.height = preprepare.Height
	prepare := &Prepare{
		PrimaryID: lbft.primaryID,
		SeqNo:     preprepare.SeqNo,
		Height:    preprepare.Height,
		OptHash:   lbft.options.Hash(),
		Digest:    digest,
		Quorum:    lbft.Quorum(),
		Chain:     lbft.options.Chain,
		ReplicaID: lbft.options.ID,
	}

	log.Debugf("Replica %s send Prepare for consensus %s", lbft.options.ID, prepare.Digest)
	lbft.recvPrepare(prepare)
	lbft.broadcast(core.fromChain, &Message{Type: MESSAGEPREPARE, Payload: utils.Serialize(prepare)})
	if core.fromChain != core.toChain {
		lbft.broadcast(core.toChain, &Message{Type: MESSAGEPREPARE, Payload: utils.Serialize(prepare)})
	}
	return nil
}

func (lbft *Lbft) recvPrepare(prepare *Prepare) *Message {
	if _, ok := lbft.committedRequests[prepare.SeqNo]; ok || prepare.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Prepare from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, prepare.ReplicaID, prepare.Digest, prepare.SeqNo, lbft.execSeqNo)
		return nil
	}

	core := lbft.getlbftCore(prepare.Digest)
	if core.prePrepare != nil && core.fromChain != prepare.Chain && core.toChain != prepare.Chain {
		log.Errorf("Replica %s received Prepare from %s for consensus %s :  ignore, illegal (%s==%s || %s==%s )", lbft.options.ID, prepare.ReplicaID, prepare.Digest, prepare.Chain, core.fromChain, prepare.Chain, core.toChain)
		return nil
	}
	for _, p := range core.prepare {
		if p.Chain == prepare.Chain && p.ReplicaID == prepare.ReplicaID {
			log.Errorf("Replica %s received Prepare from %s for consensus %s :  ignore, duplicate", lbft.options.ID, prepare.ReplicaID, prepare.Digest)
			return nil
		}
	}

	log.Debugf("Replica %s received Prepare from %s for consensus %s", lbft.options.ID, prepare.ReplicaID, prepare.Digest)

	lbft.startNewViewTimerForCore(core)
	core.prepare = append(core.prepare, prepare)
	if core.passPrepare || !lbft.maybePassPrepare(core) {
		return nil
	}
	core.passPrepare = true
	commit := &Commit{
		PrimaryID: lbft.primaryID,
		SeqNo:     core.prePrepare.SeqNo,
		Height:    core.prePrepare.Height,
		OptHash:   lbft.options.Hash(),
		Digest:    prepare.Digest,
		Quorum:    lbft.Quorum(),
		Chain:     lbft.options.Chain,
		ReplicaID: lbft.options.ID,
	}

	log.Debugf("Replica %s send Commit for consensus %s", lbft.options.ID, commit.Digest)
	lbft.recvCommit(commit)
	lbft.broadcast(core.fromChain, &Message{Type: MESSAGECOMMIT, Payload: utils.Serialize(commit)})
	if core.fromChain != core.toChain {
		lbft.broadcast(core.toChain, &Message{Type: MESSAGECOMMIT, Payload: utils.Serialize(commit)})
	}
	return nil
}

func (lbft *Lbft) recvCommit(commit *Commit) *Message {
	if _, ok := lbft.committedRequests[commit.SeqNo]; ok || commit.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Commit from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, commit.ReplicaID, commit.Digest, commit.SeqNo, lbft.execSeqNo)
		return nil
	}

	core := lbft.getlbftCore(commit.Digest)
	if core.prePrepare != nil && core.fromChain != commit.Chain && core.toChain != commit.Chain {
		log.Errorf("Replica %s received Commit from %s for consensus %s :  ignore, illegal (%s==%s || %s==%s )", lbft.options.ID, commit.ReplicaID, commit.Digest, commit.Chain, core.fromChain, commit.Chain, core.toChain)
		return nil
	}
	for _, p := range core.commit {
		if p.Chain == commit.Chain && p.ReplicaID == commit.ReplicaID {
			log.Errorf("Replica %s received Commit from %s for consensus %s :  ignore, duplicate", lbft.options.ID, commit.ReplicaID, commit.Digest)
			return nil
		}
	}

	log.Debugf("Replica %s received Commit from %s for consensus %s", lbft.options.ID, commit.ReplicaID, commit.Digest)

	lbft.stopNewViewTimerForCore(core)
	core.commit = append(core.commit, commit)
	if core.passCommit || !lbft.maybePassCommit(core) {
		return nil
	}
	core.passCommit = true
	core.prePrepare.Request.Height = commit.Height
	committed := &Committed{
		SeqNo:     core.prePrepare.SeqNo,
		Request:   core.prePrepare.Request,
		Chain:     lbft.options.Chain,
		ReplicaID: lbft.options.ID,
	}

	log.Debugf("Replica %s send Committed for consensus %s", lbft.options.ID, commit.Digest)
	lbft.recvCommitted(committed)
	lbft.broadcast(lbft.options.Chain, &Message{Type: MESSAGECOMMITTED, Payload: utils.Serialize(committed)})
	return nil
}

func (lbft *Lbft) recvCommitted(committed *Committed) *Message {
	if committed.Chain != lbft.options.Chain {
		log.Debugf("Replica %s received Committed from %s for consensus %s: ignore diff chain", lbft.options.ID, committed.ReplicaID, committed.Request.Name())
		return nil
	}
	if _, ok := lbft.committedRequests[committed.SeqNo]; ok || committed.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Committed from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, committed.ReplicaID, committed.Request.Name(), committed.SeqNo, lbft.execSeqNo)
		return nil
	}

	//log.Debugf("Replica %s received Committed from %s for consensus %s", lbft.options.ID, committed.ReplicaID, committed.Request.Name())

	// if committed.ReplicaID == lbft.options.ID {
	// 	//lbft.committedRequests[committed.SeqNo] = committed.Request
	// } else {
	fetched := []*Committed{}
	for _, c := range lbft.fetched {
		if c.SeqNo == committed.SeqNo && c.ReplicaID == committed.ReplicaID {
			continue
		}
		if c.SeqNo > lbft.execSeqNo {
			fetched = append(fetched, c)
		}
	}
	lbft.fetched = fetched
	lbft.fetched = append(lbft.fetched, committed)

	q := 0
	for _, c := range lbft.fetched {
		if c.SeqNo == committed.SeqNo {
			q++
		}
	}
	digest := committed.Request.Name()
	log.Debugf("Replica %s received Committed from %s for consensus %s, vote: %d/%d", lbft.options.ID, committed.ReplicaID, digest, lbft.Quorum(), q)
	if q >= lbft.Quorum() {
		//lbft.committedRequests[committed.SeqNo] = committed.Request
	} else {
		return nil
	}
	// }
	lbft.committedRequests[committed.SeqNo] = committed.Request
	if core, ok := lbft.coreStore[digest]; ok {
		lbft.stopNewViewTimerForCore(core)
		delete(lbft.coreStore, digest)
	}
	log.Debugf("Replica %s execute for consensus %s: seqNo:%d height:%d", lbft.options.ID, committed.Request.Name(), committed.SeqNo, committed.Request.Height)
	lbft.execute()

	// for _, core := range lbft.coreStore {
	// 	if core.prePrepare != nil {
	// 		preprepare := core.prePrepare
	// 		if preprepare.SeqNo <= lbft.execSeqNo {
	// 			lbft.stopNewViewTimerForCore(core)
	// 			delete(lbft.coreStore, core.digest)
	// 		}
	// 	} else if len(core.prepare) > 0 {
	// 		prepare := core.prepare[0]
	// 		if prepare.SeqNo <= lbft.execSeqNo {
	// 			lbft.stopNewViewTimerForCore(core)
	// 			delete(lbft.coreStore, core.digest)
	// 		}
	// 	} else if len(core.commit) > 0 {
	// 		commit := core.commit[0]
	// 		if commit.SeqNo <= lbft.execSeqNo {
	// 			lbft.stopNewViewTimerForCore(core)
	// 			delete(lbft.coreStore, core.digest)
	// 		}
	// 	}
	// }
	return nil
}

type Uint32Slice []uint32

func (us Uint32Slice) Len() int {
	return len(us)
}
func (us Uint32Slice) Less(i, j int) bool {
	return us[i] < us[j]
}
func (us Uint32Slice) Swap(i, j int) {
	us[i], us[j] = us[j], us[i]
}

func (lbft *Lbft) execute() {
	keys := Uint32Slice{}
	for seqNo := range lbft.committedRequests {
		keys = append(keys, seqNo)
	}
	sort.Sort(keys)

	nextExec := lbft.execSeqNo + 1
	for seqNo, request := range lbft.committedRequests {
		if nextExec-seqNo > uint32(lbft.options.K*3) {
			delete(lbft.committedRequests, seqNo)
		} else if seqNo == nextExec {
			if lbft.execHeight+1 != request.Height {
				lbft.execHeight = request.Height
				lbft.processBlock(lbft.outputTxs, lbft.seqNos, fmt.Sprintf("size %d", lbft.options.BlockSize))
				lbft.outputTxs = nil
				lbft.seqNos = nil
			}
			lbft.execSeqNo = nextExec
			if lbft.outputTxs.Len() == 0 {
				lbft.blockTimer.Reset(lbft.options.BlockTimeout)
			}
			lbft.outputTxs = append(lbft.outputTxs, request.Txs...)
			lbft.seqNos = append(lbft.seqNos, seqNo)
			if request.Func != nil {
				request.Func(3, request.Txs)
			}
			if request.ID == EMPTYREQUEST {
				lbft.execHeight = request.Height
				lbft.processBlock(lbft.outputTxs, lbft.seqNos, fmt.Sprintf("timeout %d", lbft.options.BlockTimeout))
				lbft.outputTxs = nil
				lbft.seqNos = nil
			}
			nextExec = seqNo + 1
		} else if seqNo > nextExec {
			if seqNo-nextExec > uint32(lbft.options.K) {
				log.Debugf("Replica %s need seqNo %d ", lbft.options.ID, nextExec)
				for n, r := range lbft.committedRequests {
					log.Debugf("Replica %s seqNo %d : %s", lbft.options.ID, n, r.Name)
				}
				log.Panicf("Replica %s fallen behind over %d", lbft.options.ID, lbft.options.K)
			}
			log.Warnf("Replica %s fetch committed %d ", lbft.options.ID, nextExec)
			fc := &FetchCommitted{
				ReplicaID: lbft.options.ID,
				Chain:     lbft.options.Chain,
				SeqNo:     nextExec,
			}
			lbft.broadcast(lbft.options.Chain, &Message{Type: MESSAGEFETCHCOMMITTED, Payload: utils.Serialize(fc)})
			break
		}
	}
}

func (lbft *Lbft) processBlock(txs types.Transactions, seqNos []uint32, reason string) {
	lbft.blockTimer.Stop()
	if len(seqNos) != 0 {
		log.Infof("Replica %s write block %d (%d transactions)  %v : %s", lbft.options.ID, lbft.execHeight, len(txs), seqNos, reason)
		lbft.outputTxsChan <- &consensus.OutputTxs{Txs: txs, SeqNos: seqNos, Time: txs[len(txs)-1].CreateTime(), Height: lbft.execHeight}
	} else {
		panic("unreachable")
	}
}

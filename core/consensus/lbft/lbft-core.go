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
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/base/log"
)

func merkleRootHash(txs []*types.Transaction) crypto.Hash {
	if len(txs) > 0 {
		hashs := make([]crypto.Hash, 0)
		for _, tx := range txs {
			hashs = append(hashs, tx.Hash())
		}
		return crypto.ComputeMerkleHash(hashs)[0]
	}
	return crypto.Hash{}
}

type lbftCore struct {
	digest       string
	txs          types.Transactions
	errTxs       types.Transactions
	prePrepare   *PrePrepare
	prepare      []*Prepare
	passPrepare  bool
	commit       []*Commit
	passCommit   bool
	newViewTimer *time.Timer

	startTime time.Time
	endTime   time.Time
	sync.RWMutex
}

func (lbft *Lbft) getlbftCore(digest string) *lbftCore {
	core, ok := lbft.coreStore[digest]
	if ok {
		return core
	}

	core = &lbftCore{
		digest: digest,
	}
	core.startTime = time.Now()
	lbft.coreStore[digest] = core
	return core
}

func (lbft *Lbft) startNewViewTimerForCore(core *lbftCore, replica string) {
	lbft.stopNewViewTimer()
	lbft.stopNewViewTimerForCore(core)
	lbft.rwVcStore.Lock()
	defer lbft.rwVcStore.Unlock()
	for k, vcl := range lbft.vcStore {
		if strings.Contains(k, "resend") {
			continue
		}
		if /*(lbft.hasPrimary() && strings.Contains(k, "lbft")) ||*/ k == core.digest {
			vcs := []*ViewChange{}
			for _, vc := range vcl.vcs {
				if vc.ReplicaID == replica {
					continue
				}
				vcs = append(vcs, vc)
			}
			if len(vcs) == 0 {
				vcl.stop()
				delete(lbft.vcStore, core.digest)
			} else {
				vcl.vcs = vcs
			}
		}
	}

	core.Lock()
	defer core.Unlock()
	if core.newViewTimer == nil && lbft.hasPrimary() {
		core.newViewTimer = time.AfterFunc(lbft.options.Request, func() {
			core.Lock()
			defer core.Unlock()
			vc := &ViewChange{
				ID:            core.digest,
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.lastPrimaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("%s request timeout(%s)", core.digest, lbft.options.Request))
			core.newViewTimer = nil
		})
	}
}

func (lbft *Lbft) stopNewViewTimerForCore(core *lbftCore) {
	core.Lock()
	defer core.Unlock()
	if core.newViewTimer != nil {
		core.newViewTimer.Stop()
		core.newViewTimer = nil
	}
}

func (lbft *Lbft) maybePassPrepare(core *lbftCore) bool {
	q := 0
	nq := 0
	self := false
	hasPrimary := false
	for _, prepare := range core.prepare {
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
		q++
		nq = prepare.Quorum
	}
	log.Debugf("Replica %s received Prepare for consensus %s, voted: %d(%d/%d,%v)", lbft.options.ID, core.digest, len(core.prepare), q, nq, self)
	return hasPrimary && self && q >= nq
}

func (lbft *Lbft) maybePassCommit(core *lbftCore) bool {
	q := 0
	nq := 0
	self := false
	hasPrimary := false
	for _, commit := range core.commit {
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
		q++
		nq = commit.Quorum
	}
	log.Debugf("Replica %s received Commit for consensus %s, voted: %d(%d/%d,%v)", lbft.options.ID, core.digest, len(core.commit), q, nq, self)
	return hasPrimary && self && q >= nq
}

func (lbft *Lbft) recvRequest(request *Request) *Message {
	digest := request.Name()
	if lbft.isPrimary() {
		if _, ok := lbft.vcStore[digest]; ok {
			return nil
		}
		core := lbft.getlbftCore(digest)
		var txs, etxs types.Transactions
		if lbft.testing {
			txs = request.Txs
			etxs = nil
		} else {
			txs, etxs = lbft.stack.VerifyTxs(request.Txs)
		}
		core.txs = txs
		core.errTxs = etxs
		lbft.seqNo++

		log.Debugf("Replica %s received Request for consensus %s", lbft.options.ID, digest)
		request.Height = lbft.height
		preprepare := &PrePrepare{
			PrimaryID:  lbft.primaryID,
			SeqNo:      lbft.seqNo,
			Height:     lbft.height,
			OptHash:    lbft.options.Hash(),
			MerkleRoot: string(merkleRootHash(core.errTxs).Bytes()),
			//Digest:    digest,
			Quorum:    lbft.Quorum(),
			Request:   request,
			Chain:     lbft.options.Chain,
			ReplicaID: lbft.options.ID,
		}

		log.Debugf("Replica %s send PrePrepare for consensus %s", lbft.options.ID, digest)
		lbft.broadcast(lbft.options.Chain, &Message{
			Type:    MESSAGEPREPREPARE,
			Payload: utils.Serialize(preprepare),
		})
		lbft.recvPrePrepare(preprepare)
		lbft.height++
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
	if len(lbft.primaryID) == 0 && len(lbft.lastPrimaryID) == 0 {
		lbft.primaryID = preprepare.ReplicaID
		lbft.seqNo = preprepare.SeqNo
		lbft.execSeqNo = lbft.seqNo
	}
	if preprepare.ReplicaID != lbft.primaryID {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, diff primayID (%s==%s)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.PrimaryID, lbft.primaryID)
		return nil
	}

	core := lbft.getlbftCore(digest)
	if core.prePrepare != nil {
		log.Errorf("Replica %s received PrePrepare from %s for consensus %s: already exist ", lbft.options.ID, preprepare.ReplicaID, digest)
		vc := &ViewChange{
			ID:            digest,
			Priority:      lbft.priority,
			PrimaryID:     lbft.options.ID,
			SeqNo:         lbft.execSeqNo,
			Height:        lbft.execHeight,
			OptHash:       lbft.options.Hash(),
			LastPrimaryID: lbft.lastPrimaryID,
			ReplicaID:     lbft.options.ID,
			Chain:         lbft.options.Chain,
		}
		lbft.sendViewChange(vc, fmt.Sprintf("already exist"))
		return nil
	}

	if !lbft.isPrimary() {

		if preprepare.SeqNo != lbft.seqNo+1 {
			log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, wrong seqNo (%d==%d)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.SeqNo, lbft.seqNo)
			vc := &ViewChange{
				ID:            digest,
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.lastPrimaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("wrong seqNo (%d==%d)", preprepare.SeqNo, lbft.seqNo+1))
			return nil
		}
		if preprepare.Height != lbft.height {
			log.Errorf("Replica %s received PrePrepare from %s for consensus %s: ignore, wrong height (%d==%d)", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.Height, lbft.height)
			vc := &ViewChange{
				ID:            digest,
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.lastPrimaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("wrong seqNo (%d==%d)", preprepare.SeqNo, lbft.seqNo+1))
			return nil
		}
		var txs, etxs types.Transactions
		if lbft.testing {
			txs = preprepare.Request.Txs
			etxs = nil
		} else {
			txs, etxs = lbft.stack.VerifyTxs(preprepare.Request.Txs)
		}
		if !bytes.Equal(merkleRootHash(etxs).Bytes(), []byte(preprepare.MerkleRoot)) {
			log.Errorf("Replica %s received PrePrepare from %s for consensus %s: failed to verify", lbft.options.ID, preprepare.ReplicaID, digest)
			vc := &ViewChange{
				ID:            digest,
				Priority:      lbft.priority,
				PrimaryID:     lbft.options.ID,
				SeqNo:         lbft.execSeqNo,
				Height:        lbft.execHeight,
				OptHash:       lbft.options.Hash(),
				LastPrimaryID: lbft.lastPrimaryID,
				ReplicaID:     lbft.options.ID,
				Chain:         lbft.options.Chain,
			}
			lbft.sendViewChange(vc, fmt.Sprintf("failed to verify"))
			lbft.function(5, txs)
			return nil
		}
		core.txs = txs
		core.errTxs = etxs
		lbft.seqNo++
		lbft.height++
	}

	log.Debugf("Replica %s received PrePrepare from %s for consensus %s, seqNo %d", lbft.options.ID, preprepare.ReplicaID, digest, preprepare.SeqNo)

	lbft.startNewViewTimerForCore(core, preprepare.ReplicaID)
	core.prePrepare = preprepare
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
	lbft.broadcast(lbft.options.Chain, &Message{Type: MESSAGEPREPARE, Payload: utils.Serialize(prepare)})
	lbft.recvPrepare(prepare)
	return nil
}

func (lbft *Lbft) recvPrepare(prepare *Prepare) *Message {
	if _, ok := lbft.committedRequests[prepare.SeqNo]; ok || prepare.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Prepare from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, prepare.ReplicaID, prepare.Digest, prepare.SeqNo, lbft.execSeqNo)
		return nil
	}

	core := lbft.getlbftCore(prepare.Digest)
	if prepare.Chain != lbft.options.Chain {
		log.Errorf("Replica %s received Prepare from %s for consensus %s: ignore, diff chain (%s==%s)", lbft.options.ID, prepare.ReplicaID, prepare.Digest, prepare.Chain, lbft.options.Chain)
		return nil
	}

	log.Debugf("Replica %s received Prepare from %s for consensus %s", lbft.options.ID, prepare.ReplicaID, prepare.Digest)

	lbft.startNewViewTimerForCore(core, prepare.ReplicaID)
	core.prepare = append(core.prepare, prepare)
	if core.prePrepare == nil {
		log.Debugf("Replica %s received Prepare for consensus %s, voted: %d", lbft.options.ID, prepare.Digest, len(core.prepare))
		return nil
	}
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
	lbft.broadcast(lbft.options.Chain, &Message{Type: MESSAGECOMMIT, Payload: utils.Serialize(commit)})
	lbft.recvCommit(commit)
	return nil
}

func (lbft *Lbft) recvCommit(commit *Commit) *Message {
	if _, ok := lbft.committedRequests[commit.SeqNo]; ok || commit.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Commit from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, commit.ReplicaID, commit.Digest, commit.SeqNo, lbft.execSeqNo)
		return nil
	}

	core := lbft.getlbftCore(commit.Digest)
	if commit.Chain != lbft.options.Chain {
		log.Errorf("Replica %s received Commit from %s for consensus %s: ignore, diff chain (%s==%s)", lbft.options.ID, commit.ReplicaID, commit.Digest, commit.Chain, lbft.options.Chain)
		return nil
	}

	log.Debugf("Replica %s received Commit from %s for consensus %s", lbft.options.ID, commit.ReplicaID, commit.Digest)

	lbft.startNewViewTimerForCore(core, commit.ReplicaID)
	core.commit = append(core.commit, commit)
	if core.prePrepare == nil {
		log.Debugf("Replica %s received Commit for consensus %s, voted: %d", lbft.options.ID, commit.Digest, len(core.commit))
		return nil
	}
	if core.passCommit || !lbft.maybePassCommit(core) {
		return nil
	}
	lbft.stopNewViewTimerForCore(core)
	core.passCommit = true
	core.endTime = time.Now()
	committed := &Committed{
		SeqNo:     core.prePrepare.SeqNo,
		Height:    core.prePrepare.Height,
		Digest:    commit.Digest,
		Txs:       core.txs,
		ErrTxs:    core.errTxs,
		Chain:     lbft.options.Chain,
		ReplicaID: lbft.options.ID,
	}

	log.Debugf("Replica %s send Committed for consensus %s", lbft.options.ID, commit.Digest)
	lbft.broadcast(lbft.options.Chain, &Message{Type: MESSAGECOMMITTED, Payload: utils.Serialize(committed)})
	lbft.recvCommitted(committed)
	return nil
}

func (lbft *Lbft) recvCommitted(committed *Committed) *Message {
	if committed.Chain != lbft.options.Chain {
		log.Debugf("Replica %s received Committed from %s for consensus %s: ignore diff chain", lbft.options.ID, committed.ReplicaID, committed.Digest)
		return nil
	}
	if _, ok := lbft.committedRequests[committed.SeqNo]; ok || committed.SeqNo <= lbft.execSeqNo {
		log.Debugf("Replica %s received Committed from %s for consensus %s: ignore delay(%d<=%d)", lbft.options.ID, committed.ReplicaID, committed.Digest, committed.SeqNo, lbft.execSeqNo)
		return nil
	}

	digest := committed.Digest
	if committed.ReplicaID == lbft.options.ID {
		log.Debugf("Replica %s received Committed from %s for consensus %s", lbft.options.ID, committed.ReplicaID, digest)
		//lbft.committedRequests[committed.SeqNo] = committed.Request
	} else {
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
		log.Debugf("Replica %s received Committed from %s for consensus %s, vote: %d/%d", lbft.options.ID, committed.ReplicaID, digest, lbft.Quorum(), q)
		if q >= lbft.Quorum() {
			//lbft.committedRequests[committed.SeqNo] = committed.Request
		} else {
			return nil
		}
	}
	if lbft.seqNo == 0 {
		lbft.seqNo = committed.SeqNo
		lbft.execSeqNo = lbft.seqNo
	}
	lbft.committedRequests[committed.SeqNo] = committed
	d, _ := time.ParseDuration("0s")
	if core, ok := lbft.coreStore[digest]; ok {
		lbft.stopNewViewTimerForCore(core)
		delete(lbft.coreStore, digest)
		d = core.endTime.Sub(core.startTime)
		d = time.Now().Sub(core.startTime)
		if lbft.testing {
			cnt := len(core.prePrepare.Request.Txs)
			if lbft.statisticsCnt != 0 && cnt != lbft.statisticsCnt {
				var min, max, sum time.Duration
				for _, d := range lbft.statistics {
					sum = sum + d
					if min == 0 || min > d {
						min = d
					}
					if max < d {
						max = d
					}
				}
				log.Infof("testing ... txs:%d\tmin: %s, max: %s, avg: %s, size: %d", lbft.statisticsCnt, min, max, sum/time.Duration(len(lbft.statistics)), len(lbft.statistics))
				lbft.statistics = make(map[string]time.Duration)
			}
			lbft.statisticsCnt = cnt
			lbft.statistics[digest] = d
			lbft.testChan <- struct{}{}
		}
		if core.txs == nil {
			lbft.function(4, core.txs)
		}
	}
	//remove invalid ViewChange
	lbft.rwVcStore.Lock()
	keys := []string{}
	for key, vcl := range lbft.vcStore {
		if vcl.vcs[0].SeqNo > committed.SeqNo {
			continue
		}
		vcl.stop()
		keys = append(keys, key)
	}
	for _, key := range keys {
		delete(lbft.vcStore, key)
	}
	lbft.rwVcStore.Unlock()
	log.Infof("Replica %s execute for consensus %s: seqNo:%d height:%d, duration: %s", lbft.options.ID, committed.Digest, committed.SeqNo, committed.Height, d)
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
		if nextExec > seqNo && nextExec-seqNo > uint32(lbft.options.K*3) {
			delete(lbft.committedRequests, seqNo)
		} else if seqNo == nextExec {
			lbft.execSeqNo = nextExec
			if lbft.seqNo < lbft.execSeqNo {
				lbft.seqNo = lbft.execSeqNo
			}
			if lbft.height < request.Height {
				lbft.height = request.Height
			}
			if lbft.execHeight != request.Height {
				panic(fmt.Sprintf("noreachable(%d +2 == %d)", lbft.execHeight, request.Height))
			}
			lbft.function(3, request.Txs)
			lbft.function(6, request.ErrTxs)
			lbft.execHeight = request.Height + 1
			var seqNos []uint32
			seqNos = append(seqNos, seqNo)
			lbft.processBlock(request.Txs, seqNos, fmt.Sprintf("block timeout(%s), block size(%d)", lbft.options.BatchTimeout, lbft.options.BatchSize))
			nextExec = seqNo + 1
		} else if seqNo > nextExec {
			if seqNo-nextExec > uint32(lbft.options.K) {
				log.Debugf("Replica %s need seqNo %d ", lbft.options.ID, nextExec)
				for n, r := range lbft.committedRequests {
					log.Debugf("Replica %s seqNo %d : %s", lbft.options.ID, n, r.Digest)
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
		t := uint32(time.Now().Unix())
		if n := len(txs); n > 0 {
			t = txs[len(txs)-1].CreateTime()
		}
		lbft.outputTxsChan <- &consensus.OutputTxs{Txs: txs, SeqNos: seqNos, Time: t, Height: lbft.execHeight}
	} else {
		panic("unreachable")
	}
}

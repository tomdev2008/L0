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

package merge

import (
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/msgnet"
	"github.com/bocheninc/base/log"
)

// SEPARATOR separates fromChain from toChain
const SEPARATOR = "|"

// TxMerge merge transactions
type TxMerge struct {
	ledger    *ledger.Ledger
	receive   Receiver
	ticker    *time.Ticker
	backupTxs map[string]*types.Transaction
}

// NewTxMerge initialization
func NewTxMerge(ledger *ledger.Ledger) *TxMerge {
	return &TxMerge{
		ledger:    ledger,
		ticker:    time.NewTicker(config.MergeDuration),
		backupTxs: make(map[string]*types.Transaction),
	}
}

func (tm *TxMerge) setReceiver(receiver Receiver) {
	tm.receive = receiver
}

func (tm *TxMerge) start() {
	go tm.eventLoop()
}

func (tm *TxMerge) sendEvent(event Event) {

	if tm.receive != nil {
		tm.receive.ProcessEvent(event)
	}
}

func (tm *TxMerge) recvEvent(event Event) {
	switch msg := event.(type) {
	case msgnet.Message:
		tx := &types.Transaction{}
		tx.Deserialize(msg.Payload)
		tm.deleteBackupTx(tx)
		if msg.Cmd == msgnet.ChainAckMergeTxsMsg {
			broadcastAckMergeTxEvent := BroadcastAckMergeTxEvent{tx: tx}
			tm.sendEvent(broadcastAckMergeTxEvent)
		}
		log.Debugln("reciveMsgNetAckMerged", tx.Hash())
	case p2p.Msg:
		tx := &types.Transaction{}
		tx.Deserialize(msg.Payload)
		tm.deleteBackupTx(tx)
		log.Debugln("reciveP2PAckMerged", tx.Hash())
	}
}

func (tm *TxMerge) deleteBackupTx(tx *types.Transaction) {
	if len(tm.backupTxs) == 0 {
		return
	}
	delete(tm.backupTxs, tx.Hash().String())
}

func (tm *TxMerge) getBackupTxs() types.Transactions {
	if len(tm.backupTxs) == 0 {
		return nil
	}
	transactions := types.Transactions{}
	for _, tx := range tm.backupTxs {
		transactions = append(transactions, tx)
	}
	return transactions
}

func (tm *TxMerge) eventLoop() {
	for {
		select {
		case <-tm.ticker.C:
			txs, err := tm.ledger.GetMergedTransaction(uint32(config.MergeDuration / time.Second))
			if err != nil {
				log.Error("get MergeTxs err: ", err)
			}
			log.Infoln("getMergetxs len:", len(txs))
			if len(txs) != 0 {
				if err := tm.mergerTx(txs); err != nil {
					log.Error(err)
				}
			}
		}
	}

}

func (tm *TxMerge) mergerTx(txs types.Transactions) error {
	if len(txs) == 0 {
		log.Debugln("no merge transaction.")
		return nil
	}

	type amountTimeHash struct {
		amount  *big.Int
		fee     *big.Int
		txTime  uint32
		txsHash []crypto.Hash
	}

	m := make(map[string]*amountTimeHash)

	for _, tx := range txs {
		fromChain := tx.FromChain()
		toChain := tx.ToChain()
		assetID := tx.AssetID()
		amount := tx.Amount()
		txTime := tx.CreateTime()
		txHash := tx.Hash()
		fee := tx.Fee()
		key := chainCoordinatesToString(fromChain, toChain, assetID)
		if ath, ok := m[key]; ok {
			ath.amount.Add(ath.amount, amount)
			ath.fee.Add(ath.fee, fee)
			ath.txTime = txTime
			ath.txsHash = append(ath.txsHash, txHash)
		} else {
			key1 := chainCoordinatesToString(toChain, fromChain, assetID)
			if ath, ok := m[key1]; ok {
				ath.amount.Sub(ath.amount, amount)
				ath.fee.Sub(ath.fee, fee)
				ath.txTime = txTime
				ath.txsHash = append(ath.txsHash, txHash)
			} else {
				m[key] = &amountTimeHash{amount: amount, txTime: txTime, fee: fee, txsHash: []crypto.Hash{txHash}}
			}
		}
	}

	transactions := types.Transactions{}
	for k, v := range m {

		from, to, assetID := stringToChainCoordinates(k)

		if v.amount.Sign() < 0 {
			from, to = to, from
		}
		transaction := tm.maketransaction(from, to, assetID, v.amount.Abs(v.amount), v.fee, v.txTime)

		log.Infoln("mergeTxData: ", transaction.Data, " mergeTxHash: ", transaction.Hash())
		if err := tm.ledger.PutTxsHashByMergeTxHash(transaction.Hash(), v.txsHash); err != nil {
			return err
		}
		delete(m, k)
		transactions = append(transactions, transaction)

	}

	uploadPayload := NewUploadPayload(uint32(config.MaxPeers), uint32(config.MergeDuration), tm.getBackupTxs(), transactions)

	dstChainID := coordinate.HexToChainCoordinate(config.ChainID).ParentCoorinate()
	log.Debugln("uploadPayload: ", *uploadPayload, "dstChainID: ", dstChainID.String(), " maxPeer: ", config.MaxPeers)
	mergeTxEvent := TxEvent{
		msg: msgnet.Message{
			Cmd:     msgnet.ChainMergeTxsMsg,
			Payload: uploadPayload.Serialize(),
		},
		peerID:     config.PeerID,
		dstChainID: dstChainID.String(),
	}

	tm.sendEvent(mergeTxEvent)

	for _, tx := range transactions {
		tm.backupTxs[tx.Hash().String()] = tx
	}

	return nil
}

func (tm *TxMerge) maketransaction(fromchain, tochain coordinate.ChainCoordinate, assetID uint32, amount *big.Int, fee *big.Int, timeStamp uint32) *types.Transaction {
	tx := types.NewTransaction(fromchain.ParentCoorinate(), tochain.ParentCoorinate(), types.TypeMerged, uint32(0), accounts.ChainCoordinateToAddress(params.ChainID), accounts.ChainCoordinateToAddress(tochain), assetID, amount, fee, timeStamp)
	//merge transaction reused tx.Data.Signature for sender
	senderAddress := accounts.ChainCoordinateToAddress(fromchain)
	sig := &crypto.Signature{}
	copy(sig[:], senderAddress[:])
	tx.WithSignature(sig)
	return tx
}

func chainCoordinatesToString(src string, dst string, assetID uint32) string {
	return src + SEPARATOR + dst + SEPARATOR + strconv.FormatUint(uint64(assetID), 10)
}

func stringToChainCoordinates(str string) (coordinate.ChainCoordinate, coordinate.ChainCoordinate, uint32) {
	strs := strings.Split(str, SEPARATOR)
	from := coordinate.HexToChainCoordinate(strs[0])
	to := coordinate.HexToChainCoordinate(strs[1])
	assetID, _ := strconv.ParseUint(strs[2], 10, 32)
	return from, to, uint32(assetID)
}

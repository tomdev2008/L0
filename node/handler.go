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

package node

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/db"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts/keystore"
	"github.com/bocheninc/L0/core/blockchain"
	"github.com/bocheninc/L0/core/consensus"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/ledger"
	"github.com/bocheninc/L0/core/merge"
	"github.com/bocheninc/L0/core/p2p"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/L0/core/types"
	"github.com/bocheninc/L0/msgnet"
	jrpc "github.com/bocheninc/L0/rpc"
	"github.com/bocheninc/base/log"
	"github.com/bocheninc/base/rpc"
	"github.com/willf/bloom"
)

// ProtocolManager manages the protocol
type ProtocolManager struct {
	*blockchain.Blockchain
	*ledger.Ledger
	*keystore.KeyStore
	*p2p.Server
	consenter consensus.Consenter
	msgnet    msgnet.Stack
	merger    *merge.Helper
	// msgrpc     *msgnet.RpcHelper
	msgCh     chan *p2p.Msg
	isStarted bool
	highest   uint32

	filter *bloom.BloomFilter
	jobs   chan *job

	jrpcServer *rpc.Server
}

var (
	filterN       uint = 1000000
	falsePositive      = 0.000001
)

// NewProtocolManager returns a new sub protocol manager.
func NewProtocolManager(db *db.BlockchainDB, netConfig *p2p.Config,
	blockchain *blockchain.Blockchain, consenter consensus.Consenter,
	ledger *ledger.Ledger, ks *keystore.KeyStore,
	mergeConfig *merge.Config, jrpcConfig *jrpc.Config, logDir string) *ProtocolManager {
	manager := &ProtocolManager{
		KeyStore:   ks,
		Ledger:     ledger,
		Blockchain: blockchain,
		consenter:  consenter,
		msgCh:      make(chan *p2p.Msg, 100),
		Server:     p2p.NewServer(db, netConfig),
		filter:     bloom.NewWithEstimates(filterN, falsePositive),
	}
	manager.Server.Protocols = append(manager.Server.Protocols, p2p.Protocol{
		Name:    params.ProtocolName,
		Version: params.ProtocolVersion,
		Run: func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
			// add peer -> sub protocol handleshake -> handle message
			return manager.handle(peer, rw)
		},
		BaseCmd: baseMsg,
	})
	manager.msgnet = msgnet.NewMsgnet(manager.peerAddress(), netConfig.RouteAddress, manager.handleMsgnetMessage, logDir)
	manager.merger = merge.NewHelper(ledger, blockchain, manager, mergeConfig)
	manager.jrpcServer = jrpc.NewServer(manager, jrpcConfig)
	if params.MaxOccurs > 1 {
		manager.jobs = make(chan *job, 100)
		startJobs(params.MaxOccurs, manager.jobs)
	}
	//manager.msgrpc = msgnet.NewRpcHelper(manager)
	return manager
}

// Start starts a protocol server
func (pm *ProtocolManager) Start() {
	pm.Server.Start()
	pm.merger.Start()

	go jrpc.StartServer(pm.jrpcServer)
	go pm.consensusReadLoop()
	go pm.broadcastLoop()
	go func() {
		for {
			<-time.NewTicker(time.Minute).C
			pm.filter.ClearAll()
		}
	}()
}

// Sign signs data with nodekey
func (pm ProtocolManager) Sign(data []byte) (*crypto.Signature, error) {
	return pm.Server.Sign(data)
}

func (pm *ProtocolManager) handle(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	pm.handleShake(rw)

	if msg, err := rw.ReadMsg(); err == nil {
		if msg.Cmd == statusMsg {
			pm.OnStatus(msg, p)
		} else {
			return fmt.Errorf("handshake error ")
		}
		return pm.handleMsg(p, rw)
	}
	return fmt.Errorf("handshake error ")
}

func (pm *ProtocolManager) handleMsg(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	for {
		m, err := rw.ReadMsg()
		if err != nil {
			return err
		}
		switch m.Cmd {
		case statusMsg:
			return fmt.Errorf("should not appear status message")
		case getBlocksMsg:
			pm.OnGetBlocks(m, p)
		case invMsg:
			pm.OnInv(m, p)
		case txMsg:
			if pm.isStarted {
				pm.OnTx(m, p)
			}
		case blockMsg:
			pm.OnBlock(m, p)
		case getdataMsg:
			pm.OnGetData(m, p)
		case consensusMsg:
			if pm.isStarted {
				pm.OnConsensus(m, p)
			}
		case broadcastAckMergeTxsMsg:
			pm.merger.HandleLocalMsg(m)
		default:
			log.Error("Unknown message")
		}
	}
}

func (pm *ProtocolManager) handleShake(rw p2p.MsgReadWriter) {
	s := &StatusData{
		StartHeight: pm.CurrentHeight(),
		Version:     params.VersionMinor,
	}
	rw.WriteMsg(*p2p.NewMsg(statusMsg, utils.Serialize(s)))
}

// Relay relays inventory to remote peers
func (pm *ProtocolManager) Relay(inv types.IInventory) {
	var (
		inventory InvVect
		msg       *p2p.Msg
	)
	//log.Debugf("ProtocolManager Relay inventory, hash:  %+v", inv.Hash())

	switch inv.(type) {
	case *types.Transaction:
		var tx types.Transaction
		tx.Deserialize(inv.Serialize())
		if pm.filter.TestAndAdd(inv.Serialize()) {
			log.Debugf("Bloom Test is true, txHash: %+v", tx.Hash())
			return
		}

		if tx.GetType() == types.TypeMerged {
			msg = p2p.NewMsg(broadcastAckMergeTxsMsg, inv.Serialize())
			break
		}

		if pm.Blockchain.ProcessTransaction(&tx, true) {
			//inventory.Type = InvTypeTx
			//inventory.Hashes = []crypto.Hash{inv.Hash()}
			msg = p2p.NewMsg(txMsg, inv.Serialize())
		}
	case *types.Block:
		block := inv.(*types.Block)
		//if pm.filter.TestAndAdd(block.Serialize()) {
		//	log.Debugf("Bloom Test is true, BlockHash: %+v", block.Hash())
		//	return
		//}

		if pm.Blockchain.ProcessBlock(block, true) {
			inventory.Type = InvTypeBlock
			inventory.Hashes = []crypto.Hash{inv.Hash()}
			log.Debugf("Relay inventory %v", inventory)
			msg = p2p.NewMsg(invMsg, utils.Serialize(inventory))
		}
	}
	if msg != nil {
		pm.msgCh <- msg
	}
}

func (pm *ProtocolManager) consensusReadLoop() {
	for {
		select {
		case consensusData := <-pm.consenter.BroadcastConsensusChannel():
			to := consensusData.To
			if bytes.Equal(coordinate.HexToChainCoordinate(to), params.ChainID) {
				log.Debugf("Broadcast Consensus Message from %v to %v", params.ChainID, coordinate.HexToChainCoordinate(to))
				pm.msgCh <- p2p.NewMsg(consensusMsg, consensusData.Payload)
			} else {
				// broadcast message to msg-net
				data := msgnet.Message{}
				data.Cmd = msgnet.ChainConsensusMsg
				data.Payload = consensusData.Payload
				res := pm.SendMsgnetMessage(pm.peerAddress(), to, data)
				log.Debugf("Broadcast consensus message to msg-net, result: %t", res)
			}
		}
	}
}

func (pm *ProtocolManager) broadcastLoop() {
	for {
		select {
		case msg := <-pm.msgCh:
			pm.Broadcast(msg)
		}
	}
}

// OnStatus handles statusMsg
func (pm *ProtocolManager) OnStatus(m p2p.Msg, p *p2p.Peer) {
	remote := StatusData{}
	if err := utils.Deserialize(m.Payload, &remote); err != nil {
		log.Errorln("OnStatus deserialize error", err)
		return
	}
	log.Debugln("-----sync----- OnStatus", "---- From ", pm.CurrentHeight(), " To ", remote.StartHeight, " Size ", remote.StartHeight-pm.CurrentHeight())
	if pm.CurrentHeight() < remote.StartHeight {
		if remote.StartHeight > pm.highest {
			pm.highest = remote.StartHeight
		}
		getBlocks := GetBlocks{
			Version:       params.VersionMajor,
			LocatorHashes: []crypto.Hash{pm.Blockchain.CurrentBlockHash()},
			HashStop:      crypto.Hash{},
		}
		p2p.SendMessage(p.Conn, p2p.NewMsg(getBlocksMsg, utils.Serialize(getBlocks)))
	} else if !pm.isStarted {
		pm.isStarted = true
		pm.Blockchain.Start()
	}
}

// OnTx processes tx message
func (pm *ProtocolManager) OnTx(m p2p.Msg, p *p2p.Peer) {

	tx := new(types.Transaction)
	if err := tx.Deserialize(m.Payload); err != nil {
		log.Errorln("OnTx deserialize error ", err)
		return
	}

	if pm.filter.TestAndAdd(m.Payload) {
		log.Debugf("Bloom Test is true, txHash: %+v", tx.Hash())
		return
	}

	//log.Debugln("OnTx Hash=", tx.Hash(), " Nonce=", tx.Nonce())

	if params.MaxOccurs > 1 {
		pm.jobs <- &job{
			In: tx,
			Exec: func(in interface{}) {
				tx := in.(*types.Transaction)
				if pm.Blockchain.ProcessTransaction(tx, false) {
					pm.msgCh <- &m
				}
			},
		}
	} else {
		if pm.Blockchain.ProcessTransaction(tx, false) {
			pm.msgCh <- &m
		}
	}
}

// OnGetBlocks processes getblocks message
func (pm *ProtocolManager) OnGetBlocks(m p2p.Msg, peer *p2p.Peer) {
	var (
		getblocks GetBlocks
		hash      crypto.Hash
		hashes    []crypto.Hash
		inventory InvVect
		err       error
	)

	if err := utils.Deserialize(m.Payload, &getblocks); err != nil {
		log.Errorln("-----sync-----", err)
		return
	}

	// TODO ????
	for _, h := range getblocks.LocatorHashes {
		hash = h
	}

	for {
		hash, err = pm.GetNextBlockHash(hash)
		if err != nil || hash.Equal(crypto.Hash{}) {
			break
		} else {
			hashes = append(hashes, hash)
		}
	}

	if len(hashes) > 0 {
		inventory.Type = InvTypeBlock
		inventory.Hashes = hashes
		log.Debugln("-----sync----- OnGetBlocks Size ", len(hashes))
		p2p.SendMessage(peer.Conn, p2p.NewMsg(invMsg, utils.Serialize(inventory)))
	}
}

func (pm *ProtocolManager) OnBlock(m p2p.Msg, peer *p2p.Peer) {

	blk := new(types.Block)
	if err := blk.Deserialize(m.Payload); err != nil {
		log.Errorln("-----sync----- OnBlock  deserialize ", err)
		return
	}

	//if pm.filter.TestAndAdd(m.Payload) {
	//	log.Debugf("Bloom Test is true, BlockHash: %+v", blk.Hash())
	//	return
	//}

	log.Debugf("-----sync----- OnBlock %s(%d)", blk.Hash(), blk.Height())
	// p.AddFilter(m.CheckSum[:])
	if pm.CurrentHeight()+1 < blk.Height() {
		getBlocks := GetBlocks{
			Version:       params.VersionMajor,
			LocatorHashes: []crypto.Hash{pm.Blockchain.CurrentBlockHash()},
			HashStop:      crypto.Hash{},
		}
		p2p.SendMessage(peer.Conn, p2p.NewMsg(getBlocksMsg, utils.Serialize(getBlocks)))
	} else if pm.Blockchain.ProcessBlock(blk, false) {
		if !pm.isStarted && pm.CurrentHeight() == pm.highest {
			pm.isStarted = true
			pm.Blockchain.Start()
		}
	}
}

// OnInv processes inventory message
func (pm *ProtocolManager) OnInv(m p2p.Msg, peer *p2p.Peer) {
	if pm.Synced() {
		return
	}

	// TODO: parse tx inv
	var (
		inventory InvVect
		data      GetData
		hashes    []crypto.Hash
	)

	if err := utils.Deserialize(m.Payload, &inventory); err != nil {
		log.Errorln("-----sync----- Inv deserialize error", err)
		return
	}

	switch inventory.Type {
	case InvTypeTx:
		for _, h := range inventory.Hashes {
			if tx, _ := pm.GetTransaction(h); tx == nil {
				hashes = append(hashes, h)
			}
		}

		data.InvList = []InvVect{
			{
				Type: InvTypeTx,
			},
		}
	case InvTypeBlock:
		for _, h := range inventory.Hashes {
			if block, _ := pm.GetBlockByHash(h.Bytes()); block == nil {
				hashes = append(hashes, h)
			}
		}
		data.InvList = []InvVect{
			{
				Type: InvTypeBlock,
			},
		}
	}

	if len(hashes) > 0 {
		log.Debugln("-----sync----- OnInv Size ", len(inventory.Hashes))
		data.InvList[0].Hashes = hashes
		msg := p2p.NewMsg(getdataMsg, utils.Serialize(data))
		p2p.SendMessage(peer.Conn, msg)
	}
}

// OnGetData processes getdata message
func (pm *ProtocolManager) OnGetData(m p2p.Msg, peer *p2p.Peer) {
	var (
		getdata GetData
	)

	if err := utils.Deserialize(m.Payload, &getdata); err != nil {
		log.Errorln("-----sync----- OnGetData deserialize error", err)
		return
	}

	for _, inventory := range getdata.InvList {
		switch inventory.Type {
		case InvTypeBlock:
			for _, h := range inventory.Hashes {
				header, err := pm.GetBlockByHash(h.Bytes())
				if err != nil || header == nil {
					log.Errorln("-----sync----- GetBlockByHash error", err)
					break
				}
				txs, err := pm.GetTxsByBlockHash(h.Bytes(), 100)
				if err != nil {
					log.Errorln("-----sync----- GetTxsByBlockHash error", err)
					break
				}

				block := types.Block{
					Header:       header,
					Transactions: txs,
				}

				log.Debugf("-----sync----- OnGetData %s(%d)", block.Hash(), block.Height())
				p2p.SendMessage(peer.Conn, p2p.NewMsg(blockMsg, block.Serialize()))
			}
		case InvTypeTx:
			for _, h := range inventory.Hashes {
				if tx, _ := pm.GetTransaction(h); tx != nil {
					msg := p2p.NewMsg(txMsg, tx.Serialize())
					p2p.SendMessage(peer.Conn, msg)
				}
			}
		}
	}
}

// OnConsensus processes consensus message
func (pm *ProtocolManager) OnConsensus(m p2p.Msg, peer *p2p.Peer) {
	log.Debugf("Req receive consensus message %v", m.Cmd)
	pm.consenter.RecvConsensus(m.Payload) //(p.ID.String(), []byte(""), m.Payload)
}

func (pm *ProtocolManager) peerAddress() string {
	return fmt.Sprintf("%s:%s", coordinate.NewChainCoordinate(params.ChainID), pm.GetLocalPeer().ID)
}

// SendMsgnetMessage sends message to msg-net
func (pm *ProtocolManager) SendMsgnetMessage(src, dst string, msg msgnet.Message) bool {
	h := crypto.Sha256(append(msg.Serialize(), src+dst...))
	sig, err := pm.Sign(h[:])

	if err != nil {
		log.Errorln(err.Error())
		return false
	}

	if pm.msgnet != nil {
		log.Debugf("==============send data=========== cmd : %v, payload: %v ", msg.Cmd, msg.Payload)
		return pm.msgnet.Send(dst, msg.Serialize(), sig[:])
	}

	return false
}

func (pm *ProtocolManager) handleMsgnetMessage(src, dst string, payload, signature []byte) error {
	sig := crypto.Signature{}
	copy(sig[:], signature)

	if signature != nil && !sig.Validate() {
		return errors.New("msg-net signature error")
	}

	if signature != nil {
		h := crypto.Sha256(append(payload, src+dst...))
		pub, err := sig.RecoverPublicKey(h[:])
		if pub == nil || err != nil {
			log.Debug("PubilcKey verify error")
			return errors.New("PubilcKey verify error")
		}
	}

	msg := msgnet.Message{}
	msg.Deserialize(payload)
	log.Debugf("recv msg-net message type %d ", msg.Cmd)

	switch msg.Cmd {
	case msgnet.ChainConsensusMsg:
		chainID, peerID := parseID(src)
		// consensus.OnNewMessage(peerID.String(), chainID, m.Payload)
		log.Debugf("recv consensus msg from  %v:%v \n", chainID, peerID)
		pm.consenter.RecvConsensus(msg.Payload)
	case msgnet.ChainTxMsg:
		tx := &types.Transaction{}
		tx.Deserialize(msg.Payload)
		pm.Blockchain.ProcessTransaction(tx, false)
		//log.Debugln("recv transaction msg: ", tx.Hash().String())
	case msgnet.ChainMergeTxsMsg:
		fallthrough
	case msgnet.ChainAckMergeTxsMsg:
		fallthrough
	case msgnet.ChainAckedMergeTxsMsg:
		chainID, peerID := parseID(src)
		pm.merger.HandleNetMsg(msg.Cmd, chainID.String(), peerID.String(), msg)
		log.Debugf("mergeRecv cmd : %v transaction msg from message net %v:%v ,src: %v\n", msg.Cmd, chainID, peerID, src)
	case msgnet.ChainRpcMsg:
		chainID, peerID := parseID(src)
		in := new(bytes.Buffer)
		out := new(bytes.Buffer)
		in.Write(msg.Payload)
		pm.jrpcServer.ServeRequest(rpc.NewJSONServerCodec(jrpc.NewHttConn(in, out)))
		log.Debugf("remote rpc cmd : %v rpc msg rom message net %v:%v, src: %v\n", msg.Cmd, chainID, peerID, src)
		pm.SendMsgnetMessage(pm.peerAddress(), src, msgnet.Message{Cmd: msg.Cmd, Payload: out.Bytes()})
		log.Debugf("Broadcast consensus message to msg-net, result: %s", string(out.Bytes()))
	case msgnet.ChainChangeCfgMsg:
		id := strings.Split(src, ":")
		// size, _ := strconv.Atoi(string(msg.Payload))
		//pm.consenter.ChangeBlockSize(size)
		log.Debugf("change consensus config cmd : %v transaction msg from message net %v:%v ,src: %v, payload: %s", msg.Cmd, id[0], id[1], src, string(msg.Payload))
	default:
		log.Debug("not know msgnet.type...")
	}
	return nil
}

// parseID returns chainID and PeerID
func parseID(peerAddress string) (coordinate.ChainCoordinate, p2p.PeerID) {
	id := strings.Split(peerAddress, ":")
	chainID := coordinate.HexToChainCoordinate(id[0])
	if len(id) == 2 {
		peerid, _ := hex.DecodeString(id[1])
		return chainID, p2p.PeerID(peerid)
	}
	return chainID, nil
}

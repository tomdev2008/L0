// Copyright (C) 2017, Beijing Bochen Technology Co.,Ltd.  All rights reserved.
//
// This file is part of msg-net
//
// The msg-net is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The msg-net is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package peer

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"time"

	"github.com/bocheninc/msg-net/config"
	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
	"github.com/bocheninc/msg-net/net/tcp"
	pb "github.com/bocheninc/msg-net/protos"
)

//NewPeer create Peer instance
func NewPeer(id string, addresses []string, function func(srcID, dstID string, payload []byte, signature []byte) error) *Peer {
	//params verify
	return &Peer{id: id, addresses: addresses, chainMessageHandle: function}
}

//Peer Define Peer class connected to Router
type Peer struct {
	id                    string
	addresses             []string
	chainMessageHandle    func(srcID, dstID string, payload []byte, signature []byte) error
	client                *tcp.Client
	durationKeepAlive     time.Duration
	timerKeepAliveTimeout *time.Timer
	cancel                context.CancelFunc
	isConned              bool
}

//IsRunning Running or not
func (p *Peer) IsRunning() bool {
	return p.client != nil && p.client.IsConnected()
}

//Start Start peer serviceH
func (p *Peer) Start() bool {
	if p.IsRunning() {
		logger.Warnf("peer %s is already running", p.id)
		return true
	}

	if len(p.addresses) == 0 {
		logger.Errorf("peer %s not specify addresses", p.id)
		return false
	}

	//keepalive
	p.durationKeepAlive = time.Second * 15
	if d, err := time.ParseDuration(config.GetString("router.timeout.keepalive")); err == nil {
		p.durationKeepAlive = d
	} else {
		logger.Warnf("failed to parse router.timeout.keepalive, set default timeout 5s --- %v", err)
	}
	p.timerKeepAliveTimeout = time.NewTimer(2 * p.durationKeepAlive)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go p.connect(ctx)

	return true
}

//Send Send msg to Router
func (p *Peer) Send(id string, payload []byte, signature []byte) bool {
	if !p.IsRunning() {
		logger.Warnf("peer %s is alreay stopped", p.id)
		return false
	}
	if !strings.Contains(id, ":") {
		logger.Infof("broadcast all chain %s peers", id)
		id = id + ":"
	}
	chainMsg := pb.ChainMessage{SrcId: p.id, DstId: id, Payload: payload, Signature: signature}
	bytes, _ := chainMsg.Serialize()
	p.client.SendChannel() <- &pb.Message{Type: pb.Message_CHAIN_MESSAGE, Payload: bytes}

	return true
}

//Stop Stop peer service
func (p *Peer) Stop() {
	if !p.IsRunning() {
		logger.Warnf("peer %s is alreay stopped", p.id)
	}
	p.cancel()

	pr := pb.Peer{Id: p.id}
	bytes, _ := pr.Serialize()
	p.client.SendChannel() <- &pb.Message{Type: pb.Message_PEER_CLOSE, Payload: bytes}

	p.client.Disconnect()
	p.client = nil
}

//String Get Peer Information
func (p *Peer) String() string {
	m := make(map[string]interface{})
	m["id"] = p.id
	m["addresses"] = p.addresses
	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v\n", err)
	}
	return string(bytes)
}

func (p *Peer) handleMsg(conn net.Conn, channel chan<- common.IMsg, m common.IMsg) error {
	p.timerKeepAliveTimeout.Stop()

	msg := m.(*pb.Message)
	switch msg.Type {
	case pb.Message_ROUTER_CLOSE:
	case pb.Message_PEER_HELLO_ACK:
	case pb.Message_KEEPALIVE:
		p.client.SendChannel() <- &pb.Message{Type: pb.Message_KEEPALIVE_ACK, Payload: nil}
	case pb.Message_KEEPALIVE_ACK:
	case pb.Message_PEER_SYNC:
	case pb.Message_ROUTER_SYNC:
	case pb.Message_ROUTER_GET:
	case pb.Message_CHAIN_MESSAGE:
		chainMsg := &pb.ChainMessage{}
		if err := chainMsg.Deserialize(msg.Payload); err != nil {
			return err
		}
		if err := p.chainMessageHandle(chainMsg.SrcId, chainMsg.DstId, chainMsg.Payload, chainMsg.Signature); err != nil {
			return err
		}
	default:
		logger.Errorf("unsupport message type --- %v", msg.Type)
	}

	p.timerKeepAliveTimeout.Reset(2 * p.durationKeepAlive)
	return nil
}

func (p *Peer) connect(ctx context.Context) {
	duration := time.Second * 5
	if d, err := time.ParseDuration(config.GetString("router.reconnect.interval")); err == nil {
		duration = d
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.timerKeepAliveTimeout.C:
			p.isConned = false
		default:
		}

		if !p.isConned {
			for _, addr := range p.addresses {
				p.client = tcp.NewClient(addr, func() common.IMsg {
					return &pb.Message{}
				}, p.handleMsg)
				if conn := p.client.Connect(); conn != nil {
					p.isConned = true
					pr := pb.Peer{Id: p.id}
					bytes, _ := pr.Serialize()
					p.client.SendChannel() <- &pb.Message{Type: pb.Message_PEER_HELLO, Payload: bytes}
					break
				}
			}
		}
		time.Sleep(duration)
	}
}

//SetLogOut set log out path
func SetLogOut(dir string) {
	config.Set("logger.out", dir)
	logger.SetOut()
}

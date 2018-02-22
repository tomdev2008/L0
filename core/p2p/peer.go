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

package p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/core/params"
)

var (
	scheme            = "encode"
	delimiter         = "&"
	maxMsgSize uint32 = 1024 * 1024 * 100
)

const (
	VP  uint32 = 1
	NVP uint32 = 2
	ALL uint32 = VP | NVP
)

var TypeName = map[uint32]string{
	VP:  "VP",
	NVP: "NVP",
	ALL: "ALL",
}

// PeerID represents the peer identity
type PeerID []byte

func (p PeerID) String() string {
	return hex.EncodeToString(p)
}

// Peer represents a peer in blockchain
type Peer struct {
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup

	lastActiveTime time.Time
	sendChannel    chan []byte

	conn    net.Conn
	ID      PeerID
	Address string
	Type    uint32

	peerManager *PeerManager
}

func NewPeer(conn net.Conn, pm *PeerManager) *Peer {
	return &Peer{
		lastActiveTime: time.Now(),
		sendChannel:    make(chan []byte, 100),
		conn:           conn,
		peerManager:    pm,
	}
}

func (peer *Peer) Start() {
	if peer.cancel != nil {
		log.Warnf("Peer %s(%s->%s) already started.", peer.String(), peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	peer.cancel = cancel
	peer.waitGroup.Add(2)
	go peer.recv(ctx)
	go peer.send(ctx)
	log.Infoln("Peer %s(%s->%s ) Started", peer.String(), peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String())
}

func (peer *Peer) Stop() {
	if peer.cancel == nil {
		log.Warnf("Peer %s(%s->%s) already stopped.", peer.String(), peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String())
		return
	}
	peer.cancel()
	peer.waitGroup.Wait()
	log.Infoln("Peer %s(%s->%s) Stopped", peer.String(), peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String())
}

func (peer *Peer) stop() {
	peer.peerManager.remove(peer.conn)
	peer.conn.Close()
	peer.conn = nil
	peer.cancel = nil
	peer.sendChannel = make(chan []byte)
}

func (peer *Peer) SendMsg(msg []byte) error {
	select {
	case peer.sendChannel <- msg:
		return nil
	default:
		return fmt.Errorf("Peer %s(%s->%s) conn send channel fully", peer.String(), peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String())
	}
}

// String is the representation of a peer as a URL.
func (peer *Peer) String() string {
	u := url.URL{Scheme: scheme}
	u.User = url.User(peer.ID.String())
	u.Host = peer.Address
	return u.String() + delimiter + strconv.FormatUint(uint64(peer.Type), 10)
}

// GetPeerAddress returns local peer address info
func (peer *Peer) GetPeerAddress() string {
	return fmt.Sprintf("%s:%s", params.ChainID, peer.ID)
}

// ParsePeer parses a peer designator.
func (peer *Peer) ParsePeer(rawurl string) error {
	urlAndType := strings.Split(rawurl, delimiter)
	peerURL := urlAndType[0]
	typeStr := urlAndType[1]
	u, err := url.Parse(peerURL)
	if err != nil {
		return err
	}
	if u.Scheme != scheme {
		return fmt.Errorf("invalid URL scheme, want \"%s\"", scheme)
	}
	// Parse the PeerID from the user portion.
	if u.User == nil {
		return errors.New("does not contain peer ID")
	}
	id, _ := hex.DecodeString(u.User.String())
	peerType, _ := strconv.ParseUint(typeStr, 10, 64)

	peer.ID = id
	peer.Address = u.Host
	peer.Type = uint32(peerType)
	return nil
}

func (peer *Peer) recv(ctx context.Context) {
	defer peer.stop()

	defer peer.waitGroup.Done()
	peer.conn.SetReadDeadline(time.Now().Add(option.DeadLine))
	headerSize := 4
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		//head
		headerBytes := make([]byte, headerSize)
		if n, err := peer.conn.Read(headerBytes); err != nil {
			log.Errorf("%s(%s->%s) conn read header --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
			return
		} else if n != headerSize {
			err := fmt.Errorf("missing (expect %v, actual %v)", headerSize, n)
			log.Errorf("%s(%s->%s) conn read header --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
			return
		}
		//data
		dataSize := binary.LittleEndian.Uint32(headerBytes)
		if dataSize > maxMsgSize {
			err := fmt.Errorf("message too big")
			log.Errorf("%s(%s->%s) conn read header --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
			return
		}
		data := make([]byte, dataSize)
		if n, err := io.ReadFull(peer.conn, data); err != nil {
			log.Errorf("%s(%s->%s) conn read data --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
			return
		} else if uint32(n) != dataSize {
			err := fmt.Errorf("missing (expect %v, actual %v)", dataSize, n)
			log.Errorf("%s(%s->%s) conn read data --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
			return
		}
		peer.lastActiveTime = time.Now()
	}
}

func (peer *Peer) send(ctx context.Context) {
	defer peer.waitGroup.Done()
	peer.conn.SetWriteDeadline(time.Now().Add(option.DeadLine))
	headerSize := 4
	for {
		select {
		case <-ctx.Done():
			return
		case dataBytes := <-peer.sendChannel:
			//headdata
			headerBytes := make([]byte, headerSize)
			dataSize := len(dataBytes)
			binary.LittleEndian.PutUint32(headerBytes, uint32(dataSize))
			var buf bytes.Buffer
			if num, err := buf.Write(headerBytes); num != headerSize || err != nil {
				log.Errorf("%s(%s->%s) conn send header --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
				continue
			}
			if num, err := buf.Write(dataBytes); num != dataSize && err != nil {
				log.Errorf("%s(%s->%s) conn send header --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
				continue
			}
			//send
			num, err := peer.conn.Write(buf.Bytes())
			if err != nil || buf.Len() != num {
				log.Errorf("%s(%s->%s) conn send header & data --- %s", peer, peer.conn.LocalAddr().String(), peer.conn.RemoteAddr().String(), err)
				continue
			}
		default:
		}

	}
}

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
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/log"
)

// Option is the p2p network configuration
type Option struct {
	ListenAddress       string
	DeadLine            time.Duration
	PrivateKey          *crypto.PrivateKey
	ConnectTimeInterval time.Duration
	ReconnectTimes      int
	KeepAliveInterval   time.Duration
	KeepAliveTimes      int
	MaxPeers            int
	MinPeers            int
	Cores               int
	BootstrapNodes      []string
	CAPath              string
}

//option defines the default network configuration
var option = &Option{
	ListenAddress:       ":20166",
	DeadLine:            time.Second,
	PrivateKey:          nil,
	ConnectTimeInterval: 30 * time.Second,
	ReconnectTimes:      3,
	KeepAliveInterval:   15 * time.Second,
	KeepAliveTimes:      30,
	MaxPeers:            8,
	MinPeers:            3,
	Cores:               1,
	BootstrapNodes:      nil,
	CAPath:              "",
}

// Server represent a p2p network server
type Server struct {
	cancel    context.CancelFunc
	waitGroup sync.WaitGroup

	peerManager *PeerManager

	RootCertificate []byte
	Certificate     []byte
}

// NewServer return a new p2p server
func NewServer() *Server {
	srv := &Server{
		peerManager: NewPeerManager(),
	}
	return srv
}

// Start start p2p network run as goroutine
func (srv *Server) Start() {
	if srv.cancel != nil {
		log.Warnf("Server already started.")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	srv.cancel = cancel
	addrs := strings.Split(option.ListenAddress, ",")
	for _, addr := range addrs {
		srv.waitGroup.Add(1)
		go srv.listen(ctx, addr)
	}
	log.Infoln("Server Started")
}

func (srv *Server) listen(ctx context.Context, addr string) (err error) {
	defer srv.waitGroup.Done()

	var (
		listener *net.TCPListener
		naddr    *net.TCPAddr
		conn     net.Conn
	)

	if naddr, err = net.ResolveTCPAddr("tcp4", addr); err != nil {
		log.Errorf("net.ResolveTCPAddr(\"tcp4\", \"%s\") error(%v)", addr, err)
		return
	}
	if listener, err = net.ListenTCP("tcp4", naddr); err != nil {
		log.Errorf("net.ListenTCP(\"tcp4\", \"%s\") error(%v)", addr, err)
		return
	}

	defer listener.Close()
	listener.SetDeadline(time.Now().Add(option.DeadLine))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if conn, err = listener.AcceptTCP(); err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			} else {
				log.Errorf("listener.Accept(\"%s\") error(%v)", listener.Addr().String(), err)
				return
			}
		}
		// handle requests
		log.Debugf("Accept connection %s, %v", conn.RemoteAddr(), conn)
		srv.peerManager.Add(conn)
	}
}

// Stop stop p2p network
func (srv *Server) Stop() {
	if srv.cancel == nil {
		log.Warnf("Server already stopped.")
		return
	}

	srv.cancel()
	srv.waitGroup.Wait()
	srv.cancel = nil
	srv.peerManager.Stop()
	log.Infoln("Server Stopped")
}

// Broadcast broadcasts message to remote peers
func (srv *Server) Broadcast(msg []byte, tp uint32) {
	srv.peerManager.Broadcast(msg, tp)
}

func (srv *Server) Connect(peer *Peer) {
	srv.peerManager.Connect(peer)
}

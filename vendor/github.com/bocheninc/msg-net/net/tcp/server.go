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

//Package tcp supply tcp newwork objectes
package tcp

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/net/common"
)

//NewServer Create a tcp server instance
func NewServer(address string, newMsg func() common.IMsg, handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error) *Server {
	server := &Server{address: address, newMsg: newMsg, handleMsg: handleMsg}
	return server
}

//Server Define tcp server class and supply newwork services
type Server struct {
	address   string
	newMsg    func() common.IMsg
	handleMsg func(net.Conn, chan<- common.IMsg, common.IMsg) error

	connMap    map[net.Conn]*ClientConn
	cancelFunc context.CancelFunc
	//ws         *sync.WaitGroup
	sync.RWMutex
}

//IsRunning Running or not for supply services
func (ts *Server) IsRunning() bool {
	return ts.cancelFunc != nil
}

//Start Start server for supply services
func (ts *Server) Start() {
	if ts.IsRunning() {
		logger.Warnf("server %s is already running.", ts.address)
		return
	}

	logger.Debugf("server %s try to start ...", ts.address)
	addr, err := net.ResolveTCPAddr("tcp", ts.address)
	if err != nil {
		logger.Errorf("server %s failed to start --- %v", ts.address, err)
		return
	}

	logger.Debugf("server %s try to listen ...", ts.address)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logger.Errorf("server %s failed to start --- %v", ts.address, err)
		return
	}
	defer listener.Close()

	ts.connMap = make(map[net.Conn]*ClientConn)
	ctx, cancelFunc := context.WithCancel(context.Background())
	ts.cancelFunc = cancelFunc
	//ts.ws = &sync.WaitGroup{}
	logger.Infof("server %s started successfully", ts.address)

	// ts.ws.Add(1)
	// defer ts.ws.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		listener.SetDeadline(time.Now().Add(common.Deadline))
		if conn, err := listener.AcceptTCP(); err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				//timeout
			} else {
				logger.Errorf("server %s failed to accept --- %v", ts.address, err)
			}
		} else {
			logger.Debugf("server %s accept a client %s ...", ts.address, conn.RemoteAddr().String())
			cc := &ClientConn{ts: ts, conn: conn}
			ts.add(conn, cc)
			cc.handleConn(ctx)
			logger.Infof("server %s information : %s", ts.address, ts.String())
		}
	}
}

//Stop Stop server for supply services
func (ts *Server) Stop() {
	if !ts.IsRunning() {
		logger.Warnf("server %s is already stopped.", ts.address)
		return
	}

	logger.Debugf("server %s try to stop ...", ts.address)

	ts.cancelFunc()
	//ts.ws.Wait()
	ts.cancelFunc = nil
	//ts.ws = nil
	ts.connMap = nil
	logger.Infof("server %s stop successfully", ts.address)
}

//Disconnect Close connection
func (ts *Server) Disconnect(conn net.Conn) {
	tc := ts.remove(conn)
	if tc != nil {
		tc.disconnect()
	}
}

//BroadCast Broadcast msg
func (ts *Server) BroadCast(msg common.IMsg, function func(*ClientConn, common.IMsg) error) {
	ts.IterFunc(func(conn net.Conn, cc *ClientConn) {
		if err := function(cc, msg); err != nil {
			logger.Errorf("server %s failed to broadcast msg to %s  --- %v", ts.address, conn.RemoteAddr().String(), err)
		}
	})
}

//String Get tcp server information
func (ts *Server) String() string {
	m := make(map[string]interface{})

	m["address"] = ts.address
	v := make([]interface{}, 0)
	ts.IterFunc(func(conn net.Conn, cc *ClientConn) {
		v = append(v, conn.RemoteAddr().String())
	})
	m["clients_cnt"] = len(v)
	m["clients"] = v

	bytes, err := json.Marshal(m)
	if err != nil {
		logger.Errorf("failed to json marshal --- %v", err)
	}
	return string(bytes)
}

func (ts *Server) add(conn net.Conn, cc *ClientConn) {
	ts.Lock()
	defer ts.Unlock()
	ts.connMap[conn] = cc
}

func (ts *Server) remove(conn net.Conn) *ClientConn {
	ts.Lock()
	defer ts.Unlock()
	cc, ok := ts.connMap[conn]
	if ok {
		delete(ts.connMap, conn)
	}
	return cc
}

func (ts *Server) IterFunc(function func(net.Conn, *ClientConn)) {
	ts.RLock()
	defer ts.RUnlock()
	for conn, cc := range ts.connMap {
		function(conn, cc)
	}
}

// ClientConn Server connection class
type ClientConn struct {
	ts     *Server
	conn   net.Conn
	cancel context.CancelFunc
	//ws     *sync.WaitGroup
	common.Handler
}

func (cc *ClientConn) handleConn(c context.Context) {
	cc.Handler.Init()
	ctx, cancel := context.WithCancel(c)
	cc.cancel = cancel
	//cc.ws = &sync.WaitGroup{}

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-cc.SendChannel():
				if _, err := common.Send(cc.conn, msg); err != nil {
					logger.Errorf("server %s failed to send msg to client %s --- %v", cc.conn.LocalAddr().String(), cc.conn.RemoteAddr().String(), err)
				} else {
					logger.Debugf("server %s send msg to client %s --- %v", cc.conn.LocalAddr().String(), cc.conn.RemoteAddr().String(), msg)
				}
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				cc.ts.Disconnect(cc.conn)
				return
			default:
			}
			msg := cc.ts.newMsg()
			if err := common.Recv(cc.conn, msg); err != nil {
				logger.Infof("server %s receive msg error from client %s. %v", cc.conn.LocalAddr().String(), cc.conn.RemoteAddr().String(), err)
				cc.ts.Disconnect(cc.conn)
				return
			}

			logger.Debugf("server %s receive msg success from client %s.", cc.conn.LocalAddr().String(), cc.conn.RemoteAddr().String())

			go cc.ts.handleMsg(cc.conn, cc.SendChannel(), msg)
		}
	}(ctx)
}

func (cc *ClientConn) disconnect() {
	if cc.conn == nil {
		logger.Warnf("server %s already disconnected to client", cc.ts.address)
		return
	}

	logger.Debugf("server %s try to disconnect to client %s ...", cc.conn.LocalAddr().String(), cc.conn.RemoteAddr().String())
	cc.cancel()
	cc.conn.Close()
	//cc.ws.Wait()
	cc.cancel = nil
	//cc.ws = nil
}

func (cc *ClientConn) Conn() net.Conn {
	return cc.conn
}

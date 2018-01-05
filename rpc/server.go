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

package rpc

import (
	"io"
	"net"

	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/base/rpc"
)

type pmHandler interface {
	INetWorkInfo
	IBroadcast
	LedgerInterface
	AccountInterface
}

type HttpConn struct {
	in  io.Reader
	out io.Writer
}

func NewHttConn(in io.Reader, out io.Writer) *HttpConn {
	return &HttpConn{
		in:  in,
		out: out,
	}
}

func (c *HttpConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HttpConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HttpConn) Close() error                      { return nil }

func NewServer(pmHandler pmHandler) *rpc.Server {

	server := rpc.NewServer()

	server.Register(NewAccount(pmHandler))
	server.Register(NewTransaction(pmHandler))
	server.Register(NewNet(pmHandler))
	server.Register(NewLedger(pmHandler))
	server.Register(NewMonitoring())

	return server
}

// StartServer with Test instance as a service
func StartServer(server *rpc.Server, cfg *Config) {
	jrpcCfg = &Config{}
	jrpcCfg = cfg
	if cfg.Enabled == false {
		return
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", ":"+cfg.Port); err != nil {
		log.Errorf("TestServer error %+v", err)
	}
	rpc.NewHTTPServer(server, []string{"*"}).Serve(listener)
}

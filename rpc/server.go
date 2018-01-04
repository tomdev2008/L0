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

	return server
}

// StartServer with Test instance as a service
func StartServer(server *rpc.Server, option *Option) {
	if option.Enabled == false {
		return
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", ":"+option.Port); err != nil {
		log.Errorf("TestServer error %+v", err)
	}
	rpc.NewHTTPServer(server, []string{"*"}).Serve(listener)
}

// // StartServer with Test instance as a service
// func StartServer(server *rpc.Server, option *Option) {
// 	if option.Enabled == false {
// 		return
// 	}

// 	listener, err := net.Listen("tcp", ":"+option.Port)
// 	if err != nil {
// 		log.Fatal("listen error:", err)
// 	}
// 	defer listener.Close()

// 	http.Serve(listener, http.HandlerFunc(BasicAuth(func(w http.ResponseWriter, r *http.Request) {
// 		if r.URL.Path == "/" {
// 			serverCodec := jsonrpc.NewServerCodec(&HttpConn{in: r.Body, out: w})
// 			w.Header().Set("Content-type", "application/json")
// 			w.WriteHeader(http.StatusOK)
// 			fmt.Println("hahhaha")
// 			err := server.ServeRequest(serverCodec)
// 			if err != nil {
// 				log.Printf("Error while serving JSON request: %v", err)
// 				http.Error(w, "Error while serving JSON request, details have been logged.", 500)
// 				return
// 			}
// 		}
// 	}, option.User, option.PassWord)))

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
// 	}

// }

// // ViewFunc defines view method
// type ViewFunc func(http.ResponseWriter, *http.Request)

// // BasicAuth handles basic authtication
// func BasicAuth(f ViewFunc, user, passwd string) ViewFunc {
// 	if user == "" || passwd == "" {
// 		return f
// 	}
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		basicAuthPrefix := "Basic "
// 		// get request header
// 		auth := r.Header.Get("Authorization")
// 		if strings.HasPrefix(auth, basicAuthPrefix) {
// 			payload, err := base64.StdEncoding.DecodeString(
// 				auth[len(basicAuthPrefix):],
// 			)
// 			if err == nil {
// 				pair := bytes.SplitN(payload, []byte(":"), 2)
// 				if len(pair) == 2 && bytes.Equal(pair[0], []byte(user)) &&
// 					bytes.Equal(pair[1], []byte(passwd)) {
// 					f(w, r)
// 					return
// 				}
// 			}
// 		}
// 		http.Error(w, "Error user or password.", http.StatusUnauthorized)
// 	}
// }

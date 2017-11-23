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

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/p2p"
)

var (
	conn                                                    []net.Conn
	privateKeyBytes, certificateBytes, rootCertificatebytes []byte
)

const (
	pingMsg = iota + 1
	pongMsg
	handshakeMsg
	handshakeAckMsg

	statusMsg = 17
)

var nodeID string = "0005_abc" + "_" + time.Now().String()

type Msg struct {
	Cmd      uint8
	Payload  []byte
	CheckSum [4]byte
}

type StatusData struct {
	Version     uint32
	StartHeight uint32
}

func init() {
	var err error
	privateKeyBytes, err = ioutil.ReadFile("./cert/client.key")
	if err != nil {
		fmt.Printf("read file priKey error: %s ", err)
		return
	}
	certificateBytes, err = ioutil.ReadFile("./cert/client.crt")
	if err != nil {
		fmt.Printf("read file Certificate error: %s ", err)
		return
	}
	rootCertificatebytes, err = ioutil.ReadFile("./cert/client.crt")
	if err != nil {
		fmt.Printf("read file root Certificate error: %s ", err)
		return
	}
}

func listen(c net.Conn) {
	for {
		l, err := utils.ReadVarInt(c)
		if err != nil {
			panic(err)
		}
		data := make([]byte, l)
		io.ReadFull(c, data)
		m := new(Msg)
		err = utils.Deserialize(data, m)
		if err != nil {
			panic(err)
		}
		processMsg(m, c)
	}
}

func processMsg(m *Msg, c net.Conn) {
	h := crypto.Sha256(m.Payload)
	if !bytes.Equal(m.CheckSum[:], h[0:4]) {
		println("Msg check error")
		return
	}

	respMsg := new(Msg)
	switch m.Cmd {
	case pingMsg:
		respMsg = NewMsg(pongMsg, nil)
	case handshakeMsg:
		proto := &p2p.ProtoHandshake{
			Name:       "l0-base-protocol",
			Version:    "0.0.1",
			ID:         []byte(nodeID),
			SrvAddress: "",
			Type:       p2p.TypeVp,
		}
		respMsg = NewMsg(handshakeMsg, utils.Serialize(*proto))
		fmt.Println("handshakeMsg")
	case handshakeAckMsg:

		privateKey, err := crypto.ParseKey(privateKeyBytes)
		if err != nil {
			fmt.Printf("parse key error: %s", err)
		}

		// TODO　Generate random string
		sign, err := crypto.SignRsa(privateKey, []byte("random string"))
		if err != nil {
			fmt.Printf("sign rsa error: %s", err)
		}

		encHandshake := &p2p.EncHandshake{
			Signature: sign,
			Hashed:    sha256.Sum256([]byte("random string")),
			ID:        []byte(nodeID),
			Cert:      certificateBytes,
		}

		respMsg = NewMsg(handshakeAckMsg, utils.Serialize(*encHandshake))
		fmt.Println("handshakeAckMsg")
	case statusMsg:
		status := &StatusData{
			Version:     uint32(0),
			StartHeight: uint32(0),
		}
		respMsg = NewMsg(statusMsg, utils.Serialize(*status))
		fmt.Println("statusMsg")
	default:
		fmt.Println("default conn:", c)
		return
	}
	sendMsg(respMsg, c)
}

func sendMsg(m *Msg, c net.Conn) {
	data := utils.Serialize(*m)
	data = append(utils.VarInt(uint64(len(data))), data...)
	c.Write(data)
}

// NewMsg returns a new msg
func NewMsg(MsgType uint8, payload []byte) *Msg {
	Msg := &Msg{
		Cmd:     MsgType,
		Payload: payload,
	}
	h := crypto.Sha256(payload)
	copy(Msg.CheckSum[:], h[0:4])
	return Msg
}

// TCPSend sends transaction with tcp
func TCPSend(srvAddress []string) {
	for _, address := range srvAddress {
		c, err := net.Dial("tcp", address)
		if err != nil || c == nil {
			panic(err)
		}
		go listen(c)
		fmt.Println("LocalAddr:", c.LocalAddr().String(), " RemoteAddr:", c.RemoteAddr().String())
		conn = append(conn, c)
		time.Sleep(time.Second * 5)
	}
}

// Relay relays transaction to blockchain
func Relay(m *Msg) {
	data := utils.Serialize(*m)
	data = append(utils.VarInt(uint64(len(data))), data...)
	for _, c := range conn {
		c.Write(data)
	}
}

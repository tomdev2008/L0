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
	ccrypto "crypto"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/crypto/crypter"
	"github.com/bocheninc/L0/components/utils"
)

var (
	priKey       crypter.IPrivateKey
	conn         []net.Conn
	biggetNumber int32 = 1024 * 1024 * 1024
)

const (
	pingMsg = iota + 1
	pongMsg
	handshakeMsg
	handshakeAckMsg
	secMsg    = 7
	signMsg   = 8
	statusMsg = 17
)

var nodeID string = "0005_abc"

type VerifyObj struct {
	nonce    uint32
	crt      []byte
	isPassed bool
}

var handshakeState = struct {
	sync.RWMutex
	m map[net.Conn]*VerifyObj
}{m: make(map[net.Conn]*VerifyObj)}

type Msg struct {
	Cmd      uint8
	Payload  []byte
	CheckSum [4]byte
}

type StatusData struct {
	Version     uint32
	StartHeight uint32
}

type ProtoHandshake struct {
	Name       string
	Version    string
	ID         []byte
	SrvAddress string
}

type EncHandshake struct {
	ID        []byte
	Crypter   string
	PublicKey crypter.IPublicKey
	Signature []byte
	Hash      *crypto.Hash
}

type SecMsg struct {
	Cert  []byte
	Nonce uint32
}

type CertSign struct {
	Sign []byte
}

func listen(c net.Conn) {
	localNonce := getHandshakeNonce()

	handshakeState.Lock()
	if _, ok := handshakeState.m[c]; !ok {
		handshakeState.m[c] = &VerifyObj{nonce: localNonce}
	} else {
		handshakeState.m[c].nonce = localNonce
	}
	handshakeState.Unlock()

	localCrt, _ := ioutil.ReadFile("./cert/client.crt")
	sec := &SecMsg{
		Cert:  localCrt,
		Nonce: localNonce,
	}
	respMsg := NewMsg(secMsg, utils.Serialize(*sec))
	fmt.Println("secMsg")
	sendMsg(respMsg, c)

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

		handshakeState.RLock()
		isPassed := handshakeState.m[c].isPassed
		handshakeState.RUnlock()

		if isPassed {
			go processMsg(m, c)
		} else {
			processMsg(m, c)
		}
	}
}

func getHandshakeNonce() uint32 {
	rand.Seed(int64(time.Now().Nanosecond()))
	nonce := rand.Int31n(biggetNumber)
	return uint32(nonce)
}

func processMsg(m *Msg, c net.Conn) {
	h := crypto.Sha256(m.Payload)
	if !bytes.Equal(m.CheckSum[:], h[0:4]) {
		println("Msg check error")
		return
	}

	respMsg := new(Msg)
	switch m.Cmd {
	case secMsg:
		recvMsg := &SecMsg{}
		utils.Deserialize(m.Payload, recvMsg)

		ca, _ := ioutil.ReadFile("./cert/ca.crt")

		remoteCrt := recvMsg.Cert

		handshakeState.Lock()
		if _, ok := handshakeState.m[c]; !ok {
			handshakeState.m[c] = &VerifyObj{crt: remoteCrt}
		} else {
			handshakeState.m[c].crt = remoteCrt
		}
		handshakeState.Unlock()

		isPassed := verifyCrt(ca, remoteCrt)
		if !isPassed {
			fmt.Println("verify crt failed")
			return
		}
		fmt.Println("verify crt passed")

		tmpSign, err := generateSign(recvMsg.Nonce, "./cert/client.key")
		if err != nil {
			fmt.Println("generate sign failed:", err)
			return
		}
		certSign := &CertSign{
			Sign: tmpSign,
		}
		respMsg = NewMsg(signMsg, utils.Serialize(*certSign))
		fmt.Println("signMsg")
	case signMsg:
		recvSign := &CertSign{}
		utils.Deserialize(m.Payload, recvSign)

		handshakeState.RLock()
		localNonce := handshakeState.m[c].nonce
		recvCrt := handshakeState.m[c].crt
		handshakeState.RUnlock()

		isOk := verifySign(localNonce, recvCrt, recvSign.Sign)
		if !isOk {
			fmt.Println("verify sign failed")
		}
		fmt.Println("verify sign passed")

		handshakeState.Lock()
		if _, ok := handshakeState.m[c]; !ok {
			handshakeState.m[c] = &VerifyObj{isPassed: isOk}
		} else {
			handshakeState.m[c].isPassed = isOk
		}
		handshakeState.Unlock()
		return
	case pingMsg:
		respMsg = NewMsg(pongMsg, nil)
	case handshakeMsg:
		proto := &ProtoHandshake{
			Name:       "l0-base-protocol",
			Version:    "0.0.1",
			ID:         []byte(nodeID),
			SrvAddress: "",
		}
		respMsg = NewMsg(handshakeMsg, utils.Serialize(*proto))
		fmt.Println("handshakeMsg")
	case handshakeAckMsg:
		h := crypto.Sha256([]byte("random string"))
		sign, _ := crypter.MustCrypter("secp256k1").Sign(priKey, h[:])
		enc := &EncHandshake{
			Crypter:   "secp256k1",
			PublicKey: priKey.Public(),
			Signature: sign,
			Hash:      &h,
			ID:        []byte(nodeID),
		}
		respMsg = NewMsg(handshakeAckMsg, utils.Serialize(*enc))
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
	priKey, _, _ = crypter.MustCrypter("secp256k1").GenerateKey()

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

func generateSign(nonce uint32, localKeyPath string) ([]byte, error) {
	key, err := ioutil.ReadFile(localKeyPath)
	if err != nil {
		return nil, fmt.Errorf("can't read local key\n")
	}

	block, _ := pem.Decode(key)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key\n")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error: %v\n", err)
	}

	rng := crand.Reader
	message := []byte(strconv.Itoa(int(nonce)))
	hashed := sha256.Sum256(message)
	sign, err := rsa.SignPKCS1v15(rng, privateKey, ccrypto.SHA256, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("Error from signing: %v\nc", err)
	}
	return sign, nil
}

// verifyCrt use ca crt to verify remote crt
func verifyCrt(ca []byte, remoteCrt []byte) bool {
	// get remote certificate
	pRemote, _ := pem.Decode(remoteCrt)
	if pRemote == nil || pRemote.Type != "CERTIFICATE" {
		fmt.Println("failed to decode PEM block containing public key")
		return false
	}
	certRemote, err := x509.ParseCertificate(pRemote.Bytes)
	if err != nil {
		fmt.Println("error:", err)
		return false
	}

	// get ca certificate
	pCA, _ := pem.Decode(ca)
	if pCA == nil || pCA.Type != "CERTIFICATE" {
		fmt.Println("failed to decode PEM block containing public key")
		return false
	}
	certCA, err := x509.ParseCertificate(pCA.Bytes)
	if err != nil {
		fmt.Println("error:", err)
		return false
	}

	// verify
	err = certRemote.CheckSignatureFrom(certCA)
	if err != nil {
		fmt.Println("error:", err)
		return false
	}
	return true
}

func verifySign(nonce uint32, recvCrt []byte, recvSign []byte) bool {
	p, _ := pem.Decode(recvCrt)
	if p == nil || p.Type != "CERTIFICATE" {
		fmt.Println("failed to decode PEM block containing public key")
		return false
	}
	certificate, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// verify
	message := []byte(strconv.Itoa(int(nonce)))
	hashed := sha256.Sum256(message)
	err = rsa.VerifyPKCS1v15(certificate.PublicKey.(*rsa.PublicKey), ccrypto.SHA256, hashed[:], recvSign)
	if err != nil {
		fmt.Println("verify sign failed")
		return false
	}
	return true
}

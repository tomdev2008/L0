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
	"crypto/sha256"
	"io"

	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/params"
	"github.com/bocheninc/base/log"
)

var (
	baseProtocolName    = "l0-base-protocol"
	baseProtocolVersion = "0.0.1"
	protoHandshake      *ProtoHandshake
	encHandshake        *EncHandshake
)

// Protocol raw structure
type Protocol struct {
	BaseCmd uint8
	Name    string
	Version string
	Run     func(p *Peer, rw MsgReadWriter) error
}

type protoRW struct {
	Protocol
	in chan Msg
	// exit
	w io.Writer
}

func (rw *protoRW) ReadMsg() (Msg, error) {
	select {
	case msg := <-rw.in:
		return msg, nil
	}
}

func (rw *protoRW) WriteMsg(msg Msg) (int, error) {
	return SendMessage(rw.w, &msg)
}

// ProtoHandshake is protocol handshake.  implement the interface of Protocol
type ProtoHandshake struct {
	Name       string
	Version    string
	ID         []byte
	SrvAddress string
	Type       uint32
}

// GetProtoHandshake returns protocol handshake
func GetProtoHandshake() *ProtoHandshake {
	if protoHandshake == nil {
		var localPeerType uint32
		if params.Nvp {
			localPeerType = TypeNvp
		} else {
			localPeerType = TypeVp
		}

		protoHandshake = &ProtoHandshake{
			Name:       baseProtocolName,
			Version:    baseProtocolVersion,
			ID:         getPeerID(),
			SrvAddress: getPeerAddress(config.Address),
			Type:       localPeerType,
		}
	}
	return protoHandshake
}

// GetEncHandshake returns enchandshake message
func GetEncHandshake(cert, key []byte) *EncHandshake {
	if encHandshake == nil {
		privateKey, err := crypto.ParseKey(key)
		if err != nil {
			log.Errorf("parse key error: %s", err)
			return nil
		}

		// TODO　Generate random string
		sign, err := crypto.SignRsa(privateKey, []byte("random string"))
		if err != nil {
			log.Errorf("sign rsa error: %s", err)
			return nil
		}

		encHandshake = &EncHandshake{
			Signature: sign,
			Hashed:    sha256.Sum256([]byte("random string")),
			ID:        getPeerID(),
			Cert:      cert,
		}
	}
	return encHandshake
}

// serialize ProtoHandshake instance to []byte
func (proto *ProtoHandshake) serialize() []byte {
	return utils.Serialize(*proto)
}

// deserialize buffer to ProtoHandshake instance
func (proto *ProtoHandshake) deserialize(data []byte) {
	utils.Deserialize(data, proto)
}

// matchProtocol returns the result of handshake
func (proto *ProtoHandshake) matchProtocol(i interface{}) bool {
	if p, ok := i.(*ProtoHandshake); ok {
		if p.Name == proto.Name && p.Version == proto.Version {
			return true
		}
	}
	return false
}

// EncHandshake is encryption handshake. implement the interface of Protocol
type EncHandshake struct {
	ID        []byte
	Signature []byte
	Hashed    [sha256.Size]byte
	Cert      []byte
}

// matchProtocol returns the result of handshake
func (enc *EncHandshake) matchProtocol(i interface{}) bool {
	if enc != nil && len(enc.Hashed) != 0 && enc.Cert != nil {
		cert, err := crypto.ParseCrt(enc.Cert)
		if err != nil {
			log.Errorf("parseCrt error: %s", err)
			return false
		}

		if cert.IsCA {
			return true
		}

		rootCert, err := crypto.ParseCrt(i.([]byte))
		if err != nil {
			log.Errorf("parseCrt error: %s", err)
			return false
		}

		if err := crypto.VerifyCertificate(rootCert, cert); err != nil {
			log.Errorf("Verify Certificate error: %s", err)
			return false
		}

		if err := crypto.VerifySign(enc.Hashed, enc.Signature, cert); err != nil {
			log.Errorf("enc handshake error %s", err)
			return false
		}
		return true
	}
	return false
}

// serialize EncHandshake instance to []byte
func (enc *EncHandshake) serialize() []byte {
	return utils.Serialize(enc)
}

// deserialize buffer to encHandshake instance
func (enc *EncHandshake) deserialize(data []byte) {
	utils.Deserialize(data, enc)
}

func getPeerID() []byte {
	return getPeerManager().localPeer.ID[:]
}

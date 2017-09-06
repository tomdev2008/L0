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

package sm2

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"math/big"

	"github.com/bocheninc/L0/components/crypto/crypter"
	"github.com/ethereum/go-ethereum/common/math"
)

func (priv *PrivateKey) Bytes() []byte {
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

func (priv *PrivateKey) Public() crypter.IPublicKey {
	return &priv.PublicKey
}

func (pub *PublicKey) Bytes() []byte {
	return elliptic.Marshal(P256(), pub.X, pub.Y)
}

type Crypter struct {
}

func (this *Crypter) Name() string {
	return "sm2_double256"
}

func (this *Crypter) GenerateKey() (crypter.IPrivateKey, crypter.IPublicKey, error) {
	private, err := GenerateKey(P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return private, &private.PublicKey, nil
}

func (this *Crypter) Sign(privateKey crypter.IPrivateKey, message []byte) ([]byte, error) {
	sign := this.DoubleSha256(message)
	r, s, err := Sign(rand.Reader, privateKey.(*PrivateKey), sign)
	if err != nil {
		return nil, err
	}
	b, err := asn1.Marshal(sm2Signature{r, s})
	if err != nil {
		return nil, err
	}
	return b, nil
	// tsignature := &signature.Signature{
	// 	Curve: P256(),
	// 	R:     r,
	// 	S:     s,
	// }
	// return tsignature.Serialize(), nil
}

func (this *Crypter) Verify(publicKey crypter.IPublicKey, message, sig []byte) bool {
	sign := this.DoubleSha256(message)
	signature := &sm2Signature{}
	asn1.Unmarshal(sig, signature)
	// signature, err := signature.ParseSignature(sig, P256())
	// if err != nil {
	// 	panic(err)
	// }
	return Verify(publicKey.(*PublicKey), sign, signature.R, signature.S)
}

func (this *Crypter) ToPrivateKey(data []byte) crypter.IPrivateKey {
	priv := new(PrivateKey)
	priv.PublicKey.Curve = P256()
	priv.D = new(big.Int).SetBytes(data)
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(priv.D.Bytes())
	return priv
}

func (this *Crypter) ToPublicKey(data []byte) crypter.IPublicKey {
	x, y := elliptic.Unmarshal(P256(), data)
	return &PublicKey{
		Curve: P256(),
		X:     x,
		Y:     y,
	}
}

func (this *Crypter) DoubleSha256(data []byte) []byte {
	h := sha256.Sum256(data)
	h = sha256.Sum256(h[:])
	return h[:]
}

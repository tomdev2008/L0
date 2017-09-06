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
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

func TestKeyGeneration(t *testing.T) {
	p256 := P256()
	priv, err := GenerateKey(p256, rand.Reader)
	if err != nil {
		t.Errorf("error: %s", err)
		return
	}

	if !p256.IsOnCurve(priv.PublicKey.X, priv.PublicKey.Y) {
		t.Errorf("public key invalid: %s", err)
	}
}

func TestSignAndVerify(t *testing.T) {
	p256 := P256()
	priv, _ := GenerateKey(p256, rand.Reader)

	msg := []byte("testing")

	dig := sha256.Sum256(msg)
	hashed := dig[:]
	r, s, err := Sign(rand.Reader, priv, hashed)
	if err != nil {
		t.Errorf("error signing: %s", err)
		return
	}

	if !Verify(&priv.PublicKey, hashed, r, s) {
		t.Errorf("Verify failed")
	}

	msg[0] ^= 0xff
	dig = sha256.Sum256(msg)
	hashed = dig[:]
	if Verify(&priv.PublicKey, hashed, r, s) {
		t.Errorf("Verify always works!")
	}
}

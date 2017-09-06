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
package sm2c

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestSM2C_1(t *testing.T) {
	crypter := &Crypter{}
	priv, pub, err := crypter.GenerateKey()
	if err != nil {
		panic(err)
	}
	fmt.Println("org:", priv, pub)
	fmt.Println("trs:", crypter.ToPrivateKey(priv.Bytes()), crypter.ToPublicKey(pub.Bytes()))

}
func TestSM2C_2(t *testing.T) {
	crypter := &Crypter{}
	priv, pub, err := crypter.GenerateKey()
	if err != nil {
		panic(err)
	}

	msg := []byte("testing")
	sign, err := crypter.Sign(priv, msg)
	if err != nil {
		panic(err)
	}

	fmt.Println("priv", hex.EncodeToString(priv.Bytes()))
	fmt.Println("pub ", hex.EncodeToString(pub.Bytes()))
	fmt.Println("sign ", hex.EncodeToString(sign))
	if !crypter.Verify(pub, msg, sign) {
		panic("err")
	}
}

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

package keystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/bocheninc/L0/components/crypto/crypter"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/pborman/uuid"
)

type encryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Crypter string     `json:"crypter"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type plainKeyJSON struct {
	Address    string `json:"address"`
	Crypter    string `json:"crypter"`
	PrivateKey string `json:"privatekey"`
	Id         string `json:"id"`
}

type Key struct {
	Id         uuid.UUID
	Address    accounts.Address
	PrivateKey crypter.IPrivateKey
	Crypter    string
}

type keyStore interface {
	GetKey(addr accounts.Address, filename string, auth string) (*Key, error)
	StoreKey(filename string, k *Key, auth string) error
	JoinPath(filename string) string
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

type scryptParamsJSON struct {
	N     int    `json:"n"`
	R     int    `json:"r"`
	P     int    `json:"p"`
	DkLen int    `json:"dklen"`
	Salt  string `json:"salt"`
}

// Marshal key to json bytes
func (k *Key) MarshalJSON() (j []byte, err error) {
	jStruct := plainKeyJSON{
		hex.EncodeToString(k.Address[:]),
		k.Crypter,
		hex.EncodeToString(k.PrivateKey.Bytes()),
		k.Id.String(),
	}
	j, err = json.Marshal(jStruct)
	return j, err
}

// UnmarshalJSON restore key from json
func (k *Key) UnmarshalJSON(j []byte) (err error) {
	keyJSON := new(plainKeyJSON)
	err = json.Unmarshal(j, &keyJSON)
	if err != nil {
		return err
	}

	u := new(uuid.UUID)
	*u = uuid.Parse(keyJSON.Id)
	k.Id = *u
	addr, err := hex.DecodeString(keyJSON.Address)
	if err != nil {
		return err
	}
	k.Crypter = keyJSON.Crypter
	privkey, err := hex.DecodeString(keyJSON.PrivateKey)
	if err != nil {
		return err
	}

	k.Address = accounts.NewAddress(addr)
	k.PrivateKey = crypter.MustCrypter(k.Crypter).ToPrivateKey(privkey)

	return nil
}

func newKeyFromECDSA(privateKeyECDSA crypter.IPrivateKey, crypter string) *Key {
	id := uuid.NewRandom()
	return &Key{
		Id:         id,
		Address:    accounts.PublicKeyToAddress(privateKeyECDSA.Public()),
		PrivateKey: privateKeyECDSA,
		Crypter:    crypter,
	}
}

func storeNewKey(ks keyStore, rand io.Reader, auth string, crypter string) (*Key, accounts.Account, error) {
	key, err := newKey(crypter)
	if err != nil {
		return nil, accounts.Account{}, err
	}

	a := accounts.Account{
		Crypter:   crypter,
		PublicKey: key.PrivateKey.Public(),
		URL:       accounts.URL{Scheme: KeyStoreScheme, Path: ks.JoinPath(keyFileName(key.Address))},
		Address:   key.Address,
	}
	if err := ks.StoreKey(a.URL.Path, key, auth); err != nil {
		//crypto.ZeroKey((*crypto.PrivateKey)(key.PrivateKey))
		return nil, a, err
	}
	return key, a, err
}

func newKey(crypterName string) (*Key, error) {
	crypter, err := crypter.Crypter(crypterName)
	if err != nil {
		return nil, err
	}
	privateKeyECDSA, _, err := crypter.GenerateKey()
	if err != nil {
		return nil, err
	}
	return newKeyFromECDSA(privateKeyECDSA, crypterName), nil
}

func writeKeyFile(file string, content []byte) error {
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}

	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func keyFileName(keyAddr accounts.Address) string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC--%s--%s", toISO8601(ts), hex.EncodeToString(keyAddr[:]))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

package common

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/bocheninc/L0/components/crypto"
	L0Utils "github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/accounts"
	"github.com/bocheninc/L0/core/coordinate"
	"github.com/bocheninc/L0/core/types"
)

var (
	fromChain = []byte{0}
	toChain   = []byte{0}
)

const (
	langLua = "lua"
	langJS  = "js"
	invoke  = "invoke"
)

// Get Const data
func GetLangLuaTyp() string {
	return langLua
}

func GetLangJSTyp() string {
	return langJS
}

func GetLangInvoke() string {
	return invoke
}

func GetFromChain() []byte {
	return fromChain
}

func GetToChain() []byte {
	return toChain
}

// contract lang
type contractLang string

func (lang contractLang) ConvertInitTxType() uint32 {
	switch lang {
	case langLua:
		return types.TypeLuaContractInit
	case langJS:
		return types.TypeJSContractInit
	case invoke:
		return types.TypeContractInvoke
	}
	return 0
}

// tx configure
type TxConf struct {
	fromChain  []byte
	toChain    []byte
	amount     *big.Int
	fee        *big.Int
	id         int
	receiver   accounts.Address
	privKeyHex string
}

// contract config
type ContractConf struct {
	path       string
	lang       contractLang
	isGlobal   bool
	initArgs   []string
	invokeArgs []string
}

func NewContractTxConf(fromChain, toChain []byte, amount, fee *big.Int, privKeyHex string) *TxConf {
	return &TxConf{fromChain: fromChain, toChain: toChain, amount: amount, fee: fee, privKeyHex: privKeyHex}
}

func (ctc *TxConf) GetReceiver() accounts.Address {
	return ctc.receiver
}
func NewNormalTxConf(fromChain, toChain []byte, amount, fee *big.Int, id int, receiver accounts.Address, privKeyHex string) *TxConf {
	return &TxConf{fromChain: fromChain, toChain: toChain, amount: amount, fee: fee, id: id, receiver: receiver, privKeyHex: privKeyHex}
}

func NewContractConf(path, lang string, isGlobal bool, initArgs, invokeArgs []string) *ContractConf {
	return &ContractConf{path: path, lang: contractLang(lang), isGlobal: isGlobal, initArgs: initArgs, invokeArgs: invokeArgs}
}

func CreateContractTransaction(txconf *TxConf, conf *ContractConf) *types.Transaction {
	nonce := time.Now().UnixNano()
	privkey, _ := crypto.HexToECDSA(txconf.privKeyHex)
	sender := accounts.PublicKeyToAddress(*privkey.Public())

	contractSpec := new(types.ContractSpec)
	if strings.Compare(string(conf.lang), "invoke") != 0 {
		contractSpec.ContractParams = conf.initArgs
	} else {
		contractSpec.ContractParams = conf.invokeArgs
	}

	f, _ := os.Open(conf.path)
	buf, _ := ioutil.ReadAll(f)
	contractSpec.ContractCode = buf

	if !conf.isGlobal {
		var a accounts.Address
		pubBytes := []byte(sender.String() + string(buf))
		a.SetBytes(crypto.Keccak256(pubBytes[1:])[12:])
		contractSpec.ContractAddr = a.Bytes()
	}

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(txconf.fromChain),
		coordinate.NewChainCoordinate(txconf.toChain),
		conf.lang.ConvertInitTxType(),
		uint32(nonce),
		sender,
		accounts.NewAddress(contractSpec.ContractAddr),
		1,
		txconf.amount,
		txconf.fee,
		uint32(time.Now().Unix()),
	)

	tx.Payload = L0Utils.Serialize(contractSpec)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	txconf.receiver = accounts.NewAddress(contractSpec.ContractAddr)

	return tx
}

func CreateIssueTransaction(issuePriKeyHex, privKeyHex string, id int) *types.Transaction {
	var (
		fromChain = []byte{0}
		toChain   = []byte{0}
	)

	nonce := time.Now().UnixNano()

	//issue address
	issueKey, _ := crypto.HexToECDSA(issuePriKeyHex)
	issueSender := accounts.PublicKeyToAddress(*issueKey.Public())

	//receiver address
	privkey, _ := crypto.HexToECDSA(privKeyHex)
	receiver := accounts.PublicKeyToAddress(*privkey.Public())

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeIssue,
		uint32(nonce),
		issueSender,
		receiver,
		uint32(id),
		big.NewInt(10e11),
		big.NewInt(1),
		uint32(time.Now().Unix()),
	)

	issueCoin := make(map[string]interface{})
	issueCoin["id"] = id
	tx.Payload, _ = json.Marshal(issueCoin)

	sig, _ := issueKey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)

	return tx
}

func CreateNormalTransaction(txconf *TxConf, meta []byte) *types.Transaction {
	nonce := time.Now().UnixNano()
	privkey, _ := crypto.HexToECDSA(txconf.privKeyHex)
	sender := accounts.PublicKeyToAddress(*privkey.Public())

	tx := types.NewTransaction(
		coordinate.NewChainCoordinate(fromChain),
		coordinate.NewChainCoordinate(toChain),
		types.TypeAtomic,
		uint32(nonce),
		sender,
		txconf.receiver,
		uint32(txconf.id),
		txconf.amount,
		txconf.fee,
		uint32(time.Now().Unix()),
	)
	tx.Meta = meta
	securityAsset := make(map[string]interface{})
	securityAsset["id"] = uint32(txconf.id)
	tx.Payload, _ = json.Marshal(securityAsset)
	sig, _ := privkey.Sign(tx.SignHash().Bytes())
	tx.WithSignature(sig)
	return tx
}

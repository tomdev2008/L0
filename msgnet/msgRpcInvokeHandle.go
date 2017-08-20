package msgnet

import (
	"github.com/bocheninc/L0/core/accounts"
	"encoding/json"
	"github.com/bocheninc/L0/components/log"
	"github.com/bocheninc/L0/rpc"
	"github.com/twinj/uuid"
	"github.com/bocheninc/L0/components/utils"
	"github.com/bocheninc/L0/core/ledger/state"
	"github.com/bocheninc/L0/components/crypto"
	"github.com/bocheninc/L0/core/types"
)

func manuallyInvokeRpcApi( h *RpcHelper, method string, realrpcdata []byte, serializedata *requestData ) (res []byte, cmdtype uint8) {
	var result []byte
	var msgtype uint8

	switch method {
	case "Account.New":
		output := accounts.Address{}

		rpcReqParams := &struct{
			Params []accountParams
		}{
			Params: []accountParams{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the New account params")
			break
		}
		log.Debugf("The parse params data from New account are %d, %s", rpcReqParams.Params[0].AccountType, rpcReqParams.Params[0].Passphrase)
		rpcerr := h.rpcAccount.New(&rpc.AccountNewArgs{rpcReqParams.Params[0].AccountType, rpcReqParams.Params[0].Passphrase}, &output)
		log.Debugf("The output address is %s ", output.String())
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outaddress accounts.Address
			Outputerr  string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = RpcNewAccount
	case "Account.List":
		output := []string{}
		rpcerr := h.rpcAccount.List(0, &output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outaccountslist []string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		log.Debugf("The account list output are %v", output)
		result = utils.Serialize(encapOutput)
		msgtype = RpcListAccount
	case "Account.Sign":
		output := string("")
		rpcReqParams := &struct {
			Params []signParams
		}{
			Params: []signParams{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the Sign account params")
			break
		}
		log.Debugf("The parse params data from Sign account are %s, %s, %s", rpcReqParams.Params[0].OriginAddr, rpcReqParams.Params[0].Addr, rpcReqParams.Params[0].PassPhrase)
		rpcerr := h.rpcAccount.Sign(&rpc.SignTxArgs{rpcReqParams.Params[0].OriginAddr, rpcReqParams.Params[0].Addr, rpcReqParams.Params[0].PassPhrase}, &output)
		log.Debugf("The output of Sign is %s", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outsignstr string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = RpcSignAccount
	case "Transaction.Create":
		output := string("")
		rpcReqParams := &struct{
			Params []transactionCreate
		}{
			Params: []transactionCreate{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the transaction create params")
			break
		}
		log.Debugf("The parse params data from transaction create are %s, %s, %s", rpcReqParams.Params[0].FromChain, rpcReqParams.Params[0].ToChain, rpcReqParams.Params[0].Recipient)
		log.Debugf("The Payload of the transaction is %v ", rpcReqParams.Params[0].PayLoad)
		rpcerr := h.rpcTransaction.Create(&rpc.TransactionCreateArgs{
			rpcReqParams.Params[0].FromChain,
			rpcReqParams.Params[0].ToChain,
			rpcReqParams.Params[0].Recipient,
			rpcReqParams.Params[0].Nonce,
			rpcReqParams.Params[0].Amount,
			rpcReqParams.Params[0].Fee,
			rpcReqParams.Params[0].TxType,
			rpcReqParams.Params[0].PayLoad,
		}, &output)
		log.Debugf("The output of transaction create is %s", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outtransactionout string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = RpcTransCreate
	case "Transaction.Broadcast":
		rpcReqParams := &struct{
			Params []string
		} {
			Params: []string{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the transaction broadcast params")
			break
		}
		log.Debugf("The parse params data from transaction broadcast are %s", rpcReqParams.Params[0])
		output := rpc.BroadcastReply{}
		rpcerr := h.rpcTransaction.Broadcast(rpcReqParams.Params[0], &output)
		log.Debugf("The broadcast output are %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outbroadcastreps rpc.BroadcastReply
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = RpcTransBroadcast
	case "Transaction.Query":
		output := string("")
		rpcReqParams := &struct{
			Params []transactionQuery
		}{
			Params: []transactionQuery{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the transaction query params")
			break
		}
		log.Debugf("The parse params data from transaction query are %s", rpcReqParams.Params[0].ContractAddr)
		rpcerr := h.rpcTransaction.Query(&rpc.ContractQueryArgs{
			rpcReqParams.Params[0].ContractAddr,
			rpcReqParams.Params[0].ContractParams,
		}, &output)
		log.Debugf("The output of transaction query is %s", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outquerystr string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = RpcTransBroadQuery
	case "Net.GetLocalPeer":
		output := string("")
		rpcerr := h.rpcNet.GetLocalPeer("", &output)
		log.Debugf("The get local peer output string is %s", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outpeerstr string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = NetGetLocalpeer
	case "Net.GetPeers":
		output := []string{}
		rpcerr := h.rpcNet.GetPeers("", &output)
		log.Debugf("The peers output strings are %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outpeersstr []string
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = NetGetPeers
	case "Ledger.GetBalance":
		output := state.Balance{}
		rpcReqParams := &struct{
			Params []string
		}{
			Params: []string{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger getbalance params")
			break
		}
		log.Debugf("The parse params data from ledger getbalance are %s", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetBalance(rpcReqParams.Params[0], &output)
		log.Debugf("The Balance output is %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outbalance state.Balance
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = LedgerBalance
	case "Ledger.Height":
		output := uint32(0)
		rpcerr := h.rpcLedger.Height("", &output)
		log.Debugf("The output ledger height is %d ", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outheight uint32
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = LedgerHeight
	case "Ledger.GetLastBlockHash":
		output := crypto.Hash{}
		rpcerr := h.rpcLedger.GetLastBlockHash("", &output)
		log.Debugf("The ouput last block hash is %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Lastblockhash crypto.Hash
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = LastBlockHash
	case "Ledger.GetBlockHashByNumber":
		output := crypto.Hash{}
		rpcReqParams := &struct{
			Params []uint32
		}{
			Params: []uint32{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get block hash by number of params")
			break
		}
		log.Debugf("The parse params data from ledger get block hash by number are %d", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetBlockHashByNumber(rpcReqParams.Params[0], &output)
		log.Debugf("The output block hash by number is %v ", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outblockhash crypto.Hash
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = NumberForBlockHash
	case "Ledger.GetBlockByNumber":
		output := rpc.Block{}
		rpcReqParams := &struct{
			Params []uint32
		}{
			Params: []uint32{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get block by number of params")
			break
		}
		log.Debugf("The parse params data from ledger get block by number are %d", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetBlockByNumber(rpcReqParams.Params[0], &output)
		log.Debugf("The output block by number is %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outputblock rpc.Block
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		log.Debugf("The Block from number uuid is %v", serializedata.IdentityId)
		result = utils.Serialize(encapOutput)
		msgtype = NumberForBlock
	case "Ledger.GetBlockByHash":
		output := rpc.Block{}
		rpcReqParams := &struct{
			Params []string
		}{
			Params: []string{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get block by hash of params")
			break
		}
		log.Debugf("The parse params data from ledger get block by hash are %s", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetBlockByHash(rpcReqParams.Params[0], &output)
		log.Debugf("The output block by hash is %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outputblock rpc.Block
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = HashForBlock
	case "Ledger.GetTxByHash":
		output := types.Transaction{}
		rpcReqParams := &struct{
			Params []string
		}{
			Params: []string{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get Tx by hash of params")
			break
		}
		log.Debugf("The parse params data from ledger get Tx by hash are %s", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetTxByHash(rpcReqParams.Params[0], &output)
		log.Debugf("The Tx by hash is %v ", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			OutputTx types.Transaction
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = HashForTx
	case "Ledger.GetTxsByBlockHash":
		output := types.Transactions{}
		rpcReqParams := &struct{
			Params []ledgerTxsByBlockHash
		}{
			Params: []ledgerTxsByBlockHash{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get Txs by block hash of params")
			break
		}
		log.Debugf("The parse params data from ledger get Txs by block hash are %s, %d", rpcReqParams.Params[0].BlockHash, rpcReqParams.Params[0].TxType)
		rpcerr := h.rpcLedger.GetTxsByBlockHash(rpc.GetTxsByBlockHashArgs{
			rpcReqParams.Params[0].BlockHash,
			rpcReqParams.Params[0].TxType,
		}, &output)
		log.Debugf("The output Txs by block hash are %v", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct {
			Outputtxs types.Transactions
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = BlockHashForTxs
	case "Ledger.GetTxsByBlockNumber":
		output := types.Transactions{}
		rpcReqParams := &struct{
			Params []ledgerTxsByBlockNumber
		}{
			Params: []ledgerTxsByBlockNumber{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get Txs by block number of params")
			break
		}
		log.Debugf("The parse params data from ledger get Txs by block number are %d, %d", rpcReqParams.Params[0].BlockNumber, rpcReqParams.Params[0].TxType)
		rpcerr := h.rpcLedger.GetTxsByBlockNumber(rpc.GetTxsByBlockNumberArgs{
			rpcReqParams.Params[0].BlockNumber,
			rpcReqParams.Params[0].TxType,
		}, &output)
		log.Debugf("The output txs by block number is %v ", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outputtxs types.Transactions
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = BlockNumberForTxs
	case "Ledger.GetTxsByMergeTxHash":
		output := types.Transactions{}
		rpcReqParams := &struct{
			Params []string
		}{
			Params: []string{},
		}
		err := json.Unmarshal(realrpcdata, rpcReqParams)
		if err != nil {
			log.Error("cannot parse the ledger get Txs by merge tx hash of params")
			break
		}
		log.Debug("The parse params data from ledger get Txs by merge hash are %s", rpcReqParams.Params[0])
		rpcerr := h.rpcLedger.GetTxsByMergeTxHash(rpcReqParams.Params[0], &output)
		log.Debugf("The output txs by merge tx hash is %v ", output)
		var errstr string
		if rpcerr != nil {
			errstr = rpcerr.Error()
		} else {
			errstr = ""
		}
		encapOutput := struct{
			Outputtxs types.Transactions
			Outputerr string
			IdentityId uuid.Uuid
		}{
			output,
			errstr,
			serializedata.IdentityId,
		}
		result = utils.Serialize(encapOutput)
		msgtype = MergeTxHashForTxs
	default:
		log.Debug("unknown rpc type from msgnet ...")
	}

	return result, msgtype
}

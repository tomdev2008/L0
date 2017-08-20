package msgnet

import (
	"github.com/bocheninc/L0/components/log"
	"encoding/json"
	"github.com/bocheninc/L0/rpc"
)

// HandleNetMsg handle msg from msg_net
func (h *RpcHelper) HandleNetMsg(msgType uint8, chainID string, dstID string, senddata []byte) {
	data := Message{}
	data.Cmd = msgType
	data.Payload = senddata
	res := h.pmSender.SendMsgnetMessage(chainID, dstID, data)
	log.Debugf("Broadcast consensus message to msg-net, result: %t", res)
}

func (h *RpcHelper) InvokeRpcHandle(data []byte) ([]byte, uint8) {
	var result []byte
	var msgtype uint8

	serializedata := &requestData{}
	parseErr := json.Unmarshal(data, serializedata)
	if parseErr != nil {
		log.Error("occurs error when parse the uuid data ", parseErr.Error())
	}

	log.Debugf("The guuid of send data is %s ", serializedata.IdentityId.String())

	rpcData := &rpcMessage{}
	realrpcdata := serializedata.Data
	log.Debug("The realrpcdata are ", realrpcdata)
	err := json.Unmarshal(realrpcdata, rpcData)
	if  err != nil {
		log.Error("occurs error when convert the rpc data ", err.Error())
	}

	result, msgtype = manuallyInvokeRpcApi(h, rpcData.Method, realrpcdata, serializedata)

	return result, msgtype
}

// NewHelper create instance
func NewRpcHelper(pmSender pmHandler) *RpcHelper {
	h := &RpcHelper{
		pmSender: pmSender,
		rpcAccount: rpc.NewAccount(pmSender),
		rpcTransaction: rpc.NewTransaction(pmSender),
		rpcNet: rpc.NewNet(pmSender),
		rpcLedger: rpc.NewLedger(pmSender),
	}

	return h
}

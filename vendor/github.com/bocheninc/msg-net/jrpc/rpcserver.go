package jrpc

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/bocheninc/msg-net/logger"
	"github.com/bocheninc/msg-net/peer"
	"github.com/bocheninc/msg-net/util"
	"github.com/spf13/viper"
)

var (
	virtualP0 *peer.Peer
	recvchan  = make(chan []byte)
)

type message struct {
	Id      int      `json:"id"`
	ChainId string   `json:"chainId"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
}

type MsgnetMessage struct {
	Cmd     int
	Payload []byte
}

const (
	ChainRpcMsg = 105
)

func parseRpcPost(w http.ResponseWriter, request *http.Request) {
	// Read body
	b, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Unmarshal
	var msg message
	err = json.Unmarshal(b, &msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if len(msg.ChainId) == 0 {
		http.Error(w, "should specifiy the chain id", 500)
		return
	}

	output, _ := json.Marshal(struct {
		Id     int      `json:"id"`
		Method string   `json:"method"`
		Params []string `json:"params"`
	}{msg.Id, msg.Method, msg.Params})

	data := MsgnetMessage{}
	data.Cmd = ChainRpcMsg
	data.Payload = output

	sendbytes := util.Serialize(data)
	logger.Debugf("The send request  ", string(b))

	virtualP0.Send(msg.ChainId, sendbytes, nil)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for {

		select {
		case buf := <-recvchan:
			w.Header().Set("content-type", "application/json")
			w.Write(buf)
			return
		case <-time.After(time.Second * 5):
			w.Header().Set("content-type", "application/json")
			w.Write([]byte(`{"result","timeout"}`))
			return
		}
	}

}

func chainMessageHandle(srcID, dstID string, payload []byte, signature []byte) error {
	msg := &MsgnetMessage{}
	util.Deserialize(payload, msg)
	logger.Debugf("before recontruct data of response from chain msg type: %v, payload: %v", msg.Cmd, msg.Payload)
	recvchan <- msg.Payload
	recvchan = make(chan []byte)
	logger.Debugf("rpc response rom block chain  dst: %s, src: %s\n", dstID, srcID)
	return nil
}

func RunRpcServer(port, address string) {

	id := viper.GetString("router.id")
	if id == "" {
		id = fmt.Sprintf("%d", time.Now().Nanosecond())
	}
	virtualP0 = peer.NewPeer("01:"+id, []string{address}, chainMessageHandle)
	virtualP0.Start()
	http.HandleFunc("/", parseRpcPost)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		logger.Errorf("Run msgnet rpc server fail %s", err.Error())
	} else {
		logger.Debug("Run msgnet rpc server")
	}
}

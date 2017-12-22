package common

/*
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strings"
)

type Info struct {
	duration time.Duration
	reqCount  int
	errCount  int
}

type L0Client struct {
	cli *http.Client
	url    string
	info *Info
	infoChan  chan *Info
	recvchan chan []interface{}
	publishFunc func(clientID string, data map[string]interface{})
}

func NewHttpClient() *L0Client {
	return &L0Client{
		cli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1000, //MaxIdleConnections,
			},
			Timeout: time.Duration(200) * time.Second, // RequestTimeout
		},
		info: &Info{
			duration: time.Duration(0),
			reqCount:0,
			errCount:0,
		},
		infoChan: make(chan *Info, 10),
		recvchan: make(chan []interface{}, 5),
	}
}

func (cli *L0Client) SetRPCHost(url string) {
	cli.url = url
	if !strings.Contains(cli.url, "http") {
		cli.url = "http://" + cli.url
	}
}

func (cli *L0Client) SetPublishFunc(publishFunc func(clientID string, data map[string]interface{})) {
	cli.publishFunc = publishFunc
}

func (cli *L0Client) send(data, tag interface{}) {
	cli.recvchan <- []interface{}{data, tag}
}

func (cli *L0Client) handle() {
	for {
		select {
		case data := <- cli.recvchan:
			resp, err := cli.Send(data[0].(string))
			if cli.publishFunc != nil {
				cli.publishFunc("cli0", map[string]interface{}{"txHash":data[0].(string), "data": resp, "error": err})
			}
		}
	}
}

func (cli *L0Client) Send(data string) (*ResponseData, error) {
	respData := &ResponseData{Error: fmt.Sprintf("can't connect to host: %s", cli.url)}
	req, _ := http.NewRequest("POST", cli.url, bytes.NewBufferString(data))
	req.Header.Set("Content-Type", "application/json")
	if response, err := cli.cli.Do(req); err == nil {
		defer response.Body.Close()
		if body, err := ioutil.ReadAll(response.Body); err == nil {
			err = json.Unmarshal(body, &respData)
			if err != nil {
				return respData, err
			}
		}
	} else {
		return respData, err
	}

	return respData, nil
}
*/
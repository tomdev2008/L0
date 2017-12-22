package common

//import (
//	"github.com/tendermint/tmlibs/pubsub"
//	"context"
//	"sync/atomic"
//)
//
//var node = map[string]string {
//	"cli0": "http://127.0.0.1:8881",
//	"cli1": "http://127.0.0.1:8882",
//	"cli2": "http://127.0.0.1:8883",
//	"cli3": "http://127.0.0.1:8884",
//}
//
//type workInfo struct {
//	name string
//	weight int64 //tx number in rpc cli
//	isSubscribe bool
//	cls    *L0Client
//}
//
//
//func NewWorkInfo(name string, url string) *workInfo {
//	work := &workInfo{
//		name: name,
//		weight: 0,
//		isSubscribe: false,
//		cls: NewHttpClient(),
//	}
//
//	work.cls.SetRPCHost(url)
//	return work
//}
//
//type Dispatcher struct {
//	workCli  map[string]*workInfo
//	psServer *pubsub.Server
//	receiveChan chan []interface{}
//	psContext context.Context
//}
//
//var patch *Dispatcher
//func NewDispatcher() *Dispatcher {
//	patch := &Dispatcher{
//		workCli: make(map[string]*workInfo),
//		psServer: pubsub.NewServer(),
//		receiveChan: make(chan []interface{}, 20),
//		psContext: context.Background(),
//	}
//
//	return patch
//}
//
//func Send(data interface{}, tag interface{}) {
//	patch.send(data, tag)
//}
//
//func Subcribe(clientID string, query pubsub.Query, out chan interface{}) {
//	patch.subscribe(clientID, query, out)
//}
//
//func (patch *Dispatcher) subscribe(clientID string, query pubsub.Query, out chan interface{}) {
//	patch.psServer.Subscribe(patch.psContext, clientID, query, out)
//	patch.workCli[clientID].isSubscribe = true
//}
//
//func (patch *Dispatcher) publish(clientID string, data map[string]interface{}) {
//	atomic.AddInt64(&patch.workCli[clientID].weight, -1)
//	if patch.workCli[clientID].isSubscribe {
//		patch.psServer.PublishWithTags(patch.psContext, clientID, data)
//	}
//}
//
//func (patch *Dispatcher) send(data,tag interface{}) {
//	patch.receiveChan <- []interface{}{data, tag}
//}
//
//func (patch *Dispatcher) dispatch(data, tag interface{}) {
//	workcli := patch.bestClient()
//	atomic.AddInt64(&workcli.weight,1)
//	workcli.cls.send(data, tag)
//}
//
//func (patch *Dispatcher) handler() {
//	for {
//		select {
//		case data := <-patch.receiveChan:
//			patch.dispatch(data[0], data[1])
//		}
//	}
//}
//
//func (patch *Dispatcher) bestClient() *workInfo {
//	var workCli *workInfo
//	var minWeight int64
//	for _, cli := range patch.workCli {
//		if minWeight > cli.weight {
//			workCli = cli
//			minWeight = cli.weight
//		}
//	}
//
//	return workCli
//}
//
//func init() {
//	patch = NewDispatcher()
//	for key, value := range node {
//		patch.workCli[key] = NewWorkInfo(key, value)
//		patch.workCli[key].cls.SetPublishFunc(patch.publish)
//	}
//}
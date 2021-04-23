package center

import (
	"errors"

	"github.com/hysios/edgekv"
)

type CenterServer struct {
	store edgekv.CenterStore
	mq    edgekv.MessageQueue
}

var server = &CenterServer{}

func StartServer() error {
	return server.Start()
}

func OpenEdge(edgeID edgekv.EdgeID) (edgekv.CenterStore, error) {
	panic("nonimplement")
}

func (serve *CenterServer) Start() error {
	if serve.store == nil || serve.mq == nil {
		return errors.New("don't open store or message queue")
	}

	return nil
}

func (serve *CenterServer) SetStore(store edgekv.CenterStore) {
	serve.store = store
}

func (serve *CenterServer) SetMessageQueue(mq edgekv.MessageQueue) {
	serve.mq = mq
}

func SetStore(store edgekv.CenterStore) {
	server.SetStore(store)
}

func SetMessageQueue(mq edgekv.MessageQueue) {
	server.SetMessageQueue(mq)
}

func OpenCenterStore(name string) (edgekv.CenterStore, error) {

	panic("nonimplement")
}

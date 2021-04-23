package center

import (
	"errors"
	"time"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/store/redis"
)

var RedisURI = "redis://127.0.0.1:6379?db=2"

type CenterServer struct {
	store edgekv.CenterStore
	mq    edgekv.MessageQueue
}

var server = &CenterServer{}

func StartServer() error {
	return server.Start()
}

func OpenEdge(edgeID edgekv.EdgeID) (edgekv.Store, error) {
	return server.OpenEdge(edgeID), nil
}

func (serve *CenterServer) Start() error {
	var err error
	if serve.store == nil || serve.mq == nil {
		return errors.New("don't open store or message queue")
	}

	if err = serve.mq.Subscribe(edgekv.PrefixTopic+"*", nil); err != nil {
		return err
	}

	for {
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (serve *CenterServer) SetStore(store edgekv.CenterStore) {
	serve.store = store
}

func (serve *CenterServer) SetMessageQueue(mq edgekv.MessageQueue) {
	serve.mq = mq
}

func (serve *CenterServer) OpenEdge(edgeID edgekv.EdgeID) edgekv.Store {
	return serve.store.OpenEdge(edgeID)
}

func SetStore(store edgekv.CenterStore) {
	server.SetStore(store)
}

func SetMessageQueue(mq edgekv.MessageQueue) {
	server.SetMessageQueue(mq)
}

func OpenCenterStore(name string) (edgekv.CenterStore, error) {
	switch name {
	case "redis":
		store, err := edgekv.OpenStore("redis", RedisURI)
		return store.(*redis.RedisStore), err
	default:
		return nil, errors.New("nonimplement")
	}
}

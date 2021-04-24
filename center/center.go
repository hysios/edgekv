package center

import (
	"errors"
	"time"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/store/redis"
	"github.com/hysios/log"
	"github.com/r3labs/diff/v2"
)

var RedisURI = "redis://127.0.0.1:6379?db=2"

type CenterServer struct {
	store    edgekv.CenterStore
	mq       edgekv.MessageQueue
	listener edgekv.Listener
}

var server = &CenterServer{}

func StartServer() error {
	return server.Start()
}

func OpenEdge(edgeID edgekv.EdgeID) (edgekv.Database, error) {
	return server.OpenEdge(edgeID), nil
}

func (serve *CenterServer) Start() error {
	var err error

	if serve.store == nil || serve.mq == nil {
		return errors.New("don't open store or message queue")
	}

	go serve.listener.Start()

	if err = serve.mq.Subscribe("sync", func(msg edgekv.Message) error {
		switch msg.Type {
		case edgekv.CmdChangelog:
			var (
				cmdMsg   = msg.Payload.(*edgekv.MessageChangelog)
				val      interface{}
				ok       bool
				edgeId   = edgekv.EdgeID(msg.From)
				doChange diff.Change
			)
			doChange = serve.lastChange(cmdMsg.Changes)

			if val, ok = serve.store.Get(cmdMsg.Key); ok {
				diff.Patch(cmdMsg.Changes, &val)
			} else {
				val = doChange.To
			}

			fullkey := serve.store.EdgeKey(edgeId, cmdMsg.Key)
			log.Debugf("store => %s Do [%s] change from %v to %v", fullkey, doChange.Type, doChange.From, doChange.To)
			serve.store.Set(fullkey, val)
			var event = WatchEvent{
				Key:  cmdMsg.Key,
				From: edgekv.EdgeID(msg.From),
				Old:  val,
				Val:  doChange.To,
				Done: func(ok bool) {
					if ok {
					}
				},
			}

			serve.dispatch(fullkey, event)
		}
		return nil
	}); err != nil {
		return err
	}

	for {
		time.Sleep(1 * time.Second)
	}

	return nil
}

func (serve *CenterServer) Stop() error {
	return serve.listener.Close()
}

func (serve *CenterServer) lastChange(changes diff.Changelog) diff.Change {
	l := len(changes)
	return changes[l-1]
}

func (serve *CenterServer) SetStore(store edgekv.CenterStore) {
	serve.store = store
}

func (serve *CenterServer) SetMessageQueue(mq edgekv.MessageQueue) {
	serve.mq = mq
}

type WatchEvent struct {
	Key  string
	From edgekv.EdgeID
	Old  interface{}
	Val  interface{}
	Done func(bool)
}

func (serve *CenterServer) watch(prefix string, fn edgekv.ChangeFunc) {
	log.Infof("centerServer: watch '%s'", prefix)
	serve.listener.Watch(prefix, func(key string, payload interface{}) {
		var event = payload.(WatchEvent)
		event.Done(fn(event.Key, event.Old, event.Val) == nil)
	})
}

func (serve *CenterServer) WatchEdges(prefix string, fn edgekv.EdgeChangeFunc) {
	var edgesPreifx = "*:" + prefix
	log.Infof("centerServer: watch edges '%s'", edgesPreifx)

	serve.listener.Watch(edgesPreifx, func(key string, payload interface{}) {
		var event = payload.(WatchEvent)
		event.Done(fn(event.Key, edgekv.EdgeID(event.From), event.Old, event.Val) == nil)
	})
}

func (serve *CenterServer) dispatch(key string, event WatchEvent) {
	serve.listener.Dispatch(key, event)
}

func SetStore(store edgekv.CenterStore) {
	server.SetStore(store)
}

func SetMessageQueue(mq edgekv.MessageQueue) {
	server.SetMessageQueue(mq)
}

func WatchEdges(prefix string, fn edgekv.EdgeChangeFunc) {
	server.WatchEdges(prefix, fn)
}

func OpenCenterStore(name string) (edgekv.CenterStore, error) {
	switch name {
	case "redis":
		store, err := edgekv.OpenStore("redis", RedisURI)
		return store.(*redis.RedisStore), err
	default:
		return nil, edgekv.ErrNonimpement
	}
}

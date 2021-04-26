package center

import (
	"errors"

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
	done     chan struct{}
}

var server = &CenterServer{}

func StartServer() error {
	return server.Start()
}

func Stop() error {
	return server.Stop()
}

func OpenEdge(edgeID edgekv.EdgeID) (edgekv.Database, error) {
	return server.OpenEdge(edgeID), nil
}

func (serve *CenterServer) init() {
	serve.done = make(chan struct{})
}

func (serve *CenterServer) Start() error {
	var err error
	serve.init()

	if serve.store == nil || serve.mq == nil {
		return errors.New("don't open store or message queue")
	}

	go serve.listener.Start()

	// 订阅中心同步频道
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

			fullkey := serve.store.EdgeKey(edgeId, cmdMsg.Key)
			if val, ok = serve.store.Get(fullkey); ok {
				diff.Patch(cmdMsg.Changes, &val)
				log.Infof("new val %v", val)
			} else {
				val = doChange.To
			}

			log.Debugf("store => %s Do [%s] change from %v to %v", fullkey, doChange.Type, doChange.From, doChange.To)
			serve.store.Set(fullkey, val)
			var event = edgekv.WatchEvent{
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

	<-serve.done
	return nil
}

func (serve *CenterServer) Stop() error {
	serve.done <- struct{}{}
	return serve.listener.Close()
}

func (serve *CenterServer) SetStore(store edgekv.CenterStore) {
	serve.store = store
}

func (serve *CenterServer) SetMessageQueue(mq edgekv.MessageQueue) {
	serve.mq = mq
}

func (serve *CenterServer) WatchEdges(prefix string, fn edgekv.EdgeChangeFunc) {
	var edgesPreifx = "*:" + prefix
	log.Infof("centerServer: watch edges '%s'", edgesPreifx)

	serve.listener.Watch(edgesPreifx, func(key string, payload interface{}) {
		var event = payload.(edgekv.WatchEvent)
		event.Done(fn(event.Key, edgekv.EdgeID(event.From), event.Old, event.Val) == nil)
	})
}

func (serve *CenterServer) lastChange(changes diff.Changelog) diff.Change {
	l := len(changes)
	return changes[l-1]
}

func (serve *CenterServer) watch(prefix string, fn edgekv.ChangeFunc) {
	log.Infof("centerServer: watch '%s'", prefix)
	serve.listener.Watch(prefix, func(key string, payload interface{}) {
		var event = payload.(edgekv.WatchEvent)
		event.Done(fn(event.Key, event.Old, event.Val) == nil)
	})
}

func (serve *CenterServer) dispatch(key string, event edgekv.WatchEvent) {
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

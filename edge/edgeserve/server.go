package edgeserve

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/edge"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
	. "github.com/hysios/utils/response"
	"github.com/r3labs/diff/v2"
)

type Map = map[string]interface{}

type EdgeServer struct {
	http.Server
	ID edgekv.EdgeID

	store        edgekv.Store
	mq           edgekv.MessageQueue
	listener     edgekv.Listener
	bindSessions sync.Map
}

var serve = EdgeServer{}

// Start 启动 Edge 的服务器，服务器主要以下功能
// 1. 管理与 Center 数据同步，消息推送
// 2. 提供 socket 客户端调用 API
func (serve *EdgeServer) Start() error {
	var err error

	r := mux.NewRouter()
	r.HandleFunc("/key/{key}", serve.GetKey).Methods(http.MethodGet)
	r.HandleFunc("/key/{key}", serve.SetKey).Methods(http.MethodPost)
	r.HandleFunc("/keys", serve.Keys).Methods(http.MethodGet)
	r.HandleFunc("/watch/{pattern}", serve.Watch).Methods(http.MethodGet)
	r.HandleFunc("/bind_observer/{key}", serve.BindObserver).Methods(http.MethodGet)
	r.HandleFunc("/bind/{sessID}", serve.BindRead).Methods(http.MethodGet)
	r.HandleFunc("/bind/{sessID}", serve.BindReceive).Methods(http.MethodPut)

	serve.Handler = r

	if serve.mq == nil || serve.store == nil {
		return errors.New("edge_server: mq or store is missing")
	}

	if len(serve.ID) == 0 {
		return errors.New("missing EdgeID")
	}

	go serve.listener.Start()

	topic := edgekv.Edgekey(serve.ID, "sync")
	if err = serve.mq.Subscribe(topic, func(msg edgekv.Message) error {
		switch msg.Type {
		case edgekv.CmdChangelog:
			var (
				cmdMsg   = msg.Payload.(*edgekv.MessageChangelog)
				val      interface{}
				ok       bool
				doChange diff.Change
			)
			doChange = serve.lastChange(cmdMsg.Changes)

			fullkey := cmdMsg.Key
			if val, ok = serve.store.Get(fullkey); ok {
				diff.Patch(cmdMsg.Changes, &val)
			} else {
				val = doChange.To
			}

			log.Debugf("store => %s Do [%s] change from %v to %v", fullkey, doChange.Type, doChange.From, doChange.To)
			serve.store.Set(fullkey, val)
			var event = edgekv.WatchEvent{
				Key: cmdMsg.Key,
				Old: val,
				Val: doChange.To,
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
	return serve.listenUnix()
}

// GetKey 取键值
func (serve *EdgeServer) GetKey(w http.ResponseWriter, r *http.Request) {
	var (
		vars    = mux.Vars(r)
		key     = vars["key"]
		encoder func(val interface{}, q url.Values) []byte
		q       = r.URL.Query()
		val     interface{}
		ok      bool
	)

	log.Debugf("GET: %s", key)

	contentType := r.Header.Get("Content-Type")
	log.Debugf("context type %s", contentType)
	encoder = serve.encodeGob
	switch contentType {
	case mime.TypeByExtension(".json"), "":
		encoder = serve.encodeJson
	case edgekv.BinaryMimeType:
	}

	if val, ok = serve.store.Get(key); !ok {
		AbortErr(w, http.StatusNotFound, fmt.Errorf("not found key '%s'", key))
		return
	}

	b := encoder(edge.EdgeData{Status: "success", Data: val}, q)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", contentType)
	w.Write(b)
}

// SetKey 设置键值
func (serve *EdgeServer) SetKey(w http.ResponseWriter, r *http.Request) {
	var (
		vars     = mux.Vars(r)
		key      = vars["key"]
		err      error
		decoder  func([]byte, url.Values) interface{}
		q        = r.URL.Query()
		b        []byte
		old, val interface{}
	)

	contentType := r.Header.Get("Content-Type")
	log.Debugf("context type %s", contentType)
	switch contentType {
	case mime.TypeByExtension(".json"), "":
		decoder = serve.decodeJson
	case edgekv.BinaryMimeType:
		decoder = serve.decodeGob
	}

	b, _ = ioutil.ReadAll(r.Body)

	// 转码
	val = decoder(b, q)

	log.Debugf("POST: %s with value %v", key, val)
	if old, err = serve.store.Set(key, val); err != nil {
		AbortErr(w, http.StatusInternalServerError, err)
		return
	}

	if err = serve.Sync(old, val, key); err != nil {
		AbortErr(w, http.StatusInternalServerError, err)
		return
	}

	Jsonify(w, nil)
}

func (serve *EdgeServer) Keys(w http.ResponseWriter, r *http.Request) {
	var (
		encoder func(val interface{}, q url.Values) []byte
		q       = r.URL.Query()
		b       []byte
	)
	keys := serve.store.AllKeys()
	if keys == nil {
		AbortErr(w, http.StatusNotFound, fmt.Errorf("not have any keys"))
		return
	}
	log.Infof("keys %v", keys)
	contentType := r.Header.Get("Content-Type")
	log.Debugf("context type %s", contentType)
	encoder = serve.encodeGob
	switch contentType {
	case mime.TypeByExtension(".json"), "":
		encoder = serve.encodeJson
	case edgekv.BinaryMimeType:
	}

	b = encoder(edge.EdgeData{Status: "success", Data: keys}, q)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", contentType)
	w.Write(b)
}

// Watch 监听变化的键
func (sever *EdgeServer) Watch(w http.ResponseWriter, r *http.Request) {
	var (
		vars    = mux.Vars(r)
		pattern = vars["pattern"]
	)

	type change struct {
		Key    string
		Change interface{}
	}
	var chEvent = make(chan change)

	serve.watch(pattern, func(key string, old, new interface{}) error {
		log.Infof("change event key '%s' value => %v", key, new)
		chEvent <- change{Key: key, Change: new}
		return nil
	})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	f, ok := w.(http.Flusher)
	if ok {
		f.Flush()
	} else {
		log.Infof("Damn, no flush")
		return
	}

	for event := range chEvent {
		log.Infof("event %v", event)
		b, _ := utils.Marshal(edge.EdgeEvent{Key: event.Key, Change: event.Change})
		fmt.Fprintf(w, "change: %s\n\n", base64.StdEncoding.EncodeToString(b))
		f.Flush()
	}
}

// BindObserver 的监听服务
func (sever *EdgeServer) BindObserver(w http.ResponseWriter, r *http.Request) {

	var (
		vars   = mux.Vars(r)
		key    = vars["key"]
		stream *edge.MsgStream
		err    error
	)

	stream, err = edge.Upgrade(w, r)
	if err != nil {
		AbortErr(w, http.StatusBadGateway, err)
	}

	log.Debugf("Bind: %s", key)

	// topic := edgekv.Edgekey(serve.ID, "binder")
	var msg = edgekv.Message{
		From: string(serve.ID),
		Type: edgekv.CmdDeclareBinder,
		Payload: edgekv.MessageDeclareBinder{
			Pattern: key,
		},
	}

	// msg.Build()
	serve.mq.Publish("binder", msg) // tell Center observer key binded
	bindTopic := edgekv.Edgekey(serve.ID, "bind_get")

	// var msgCh = make(chan edgekv.Message)
	// subscribe bind_get topic, when center to get bind key
	go serve.mq.Subscribe(bindTopic, func(msg edgekv.Message) error {
		switch msg.Type {
		case edgekv.CmdGetBind:
			stream.SendMsg(msg)
		case edgekv.CmdSetBind:
			stream.SendMsg(msg)
		default:
			return errors.New("invalid msg type in Bind Get topic")
		}
		return nil
	})

	stream.MessageMsg(func(msg edgekv.Message) {

	})
}

func (sever *EdgeServer) BindRead(w http.ResponseWriter, r *http.Request) {
}

func (sever *EdgeServer) BindReceive(w http.ResponseWriter, r *http.Request) {
}

// Stop 停止服务
func (serve *EdgeServer) Stop() error {
	var ctx = context.Background()

	return serve.Shutdown(ctx)
}

func (serve *EdgeServer) SetEdgeID(id edgekv.EdgeID) {
	serve.ID = id
}

func (serve *EdgeServer) SetStore(store edgekv.Store) {
	serve.store = store
}

func (serve *EdgeServer) SetMessageQueue(mq edgekv.MessageQueue) {
	serve.mq = mq
}

func (serve *EdgeServer) Sync(old, val interface{}, key string) error {
	var (
		changes diff.Changelog
		err     error
	)
	switch x := old.(type) {
	case map[string]interface{}:
		if changes, err = diff.Diff(x, val); err != nil {
			changes = diff.Changelog{{
				Type: diff.DELETE, Path: []string{key},
			}, {
				Type: diff.CREATE, Path: []string{key}, To: val,
			}}
		}
	default:
		if reflect.DeepEqual(old, val) {
			return nil
		}
		if old == nil {
			changes = diff.Changelog{{Type: diff.CREATE, To: val}}
		} else {
			changes = diff.Changelog{{Type: diff.UPDATE, From: x, To: val}}
		}
	}

	if len(changes) == 0 {
		return nil
	}

	return serve.mq.Publish("sync", edgekv.Message{
		From: string(serve.ID),
		Type: edgekv.CmdChangelog,
		Payload: edgekv.MessageChangelog{
			Key:     key,
			Changes: changes,
		},
	})
}

func (serve *EdgeServer) decodeType(val interface{}, b []byte, q url.Values) error {
	var (
		s   = string(b)
		rd  = strings.NewReader(s)
		dec = json.NewDecoder(rd)
		v   = reflect.ValueOf(val)
	)
	v = reflect.Indirect(v)

	switch q.Get("type") {
	case "duration":
		if dt, err := time.ParseDuration(s); err != nil {
			return err
		} else {
			v.Set(reflect.ValueOf(dt))
		}
	case "time":
		s = unquote(s)
		if t, err := time.Parse(time.RFC3339, s); err != nil {
			return err
		} else {
			v.Set(reflect.ValueOf(t))
		}
	case "bytes":
		v.Set(reflect.ValueOf(atob(s)))
	default:
		return dec.Decode(val)
	}

	return nil
}

func (serve *EdgeServer) encodeJson(val interface{}, q url.Values) []byte {
	return []byte(utils.Stringify(val))
}

func (serve *EdgeServer) encodeGob(val interface{}, q url.Values) []byte {
	if b, err := utils.Marshal(val); err == nil {
		return b
	}
	return nil
}

func (serve *EdgeServer) decodeJson(b []byte, q url.Values) interface{} {
	var (
		typ = q.Get("type")
	)

	if len(typ) > 0 {
		var val interface{}
		serve.decodeType(&val, b, q)
		return val
	}

	return utils.JSON(string(b))
}

func (serve *EdgeServer) decodeGob(b []byte, q url.Values) interface{} {
	var val edge.EdgeData
	if err := utils.Unmarshal(b, &val); err == nil {
		return val.Data
	}
	return nil
}

func (serve *EdgeServer) listenUnix() error {
	os.Remove(edge.UnixSock)

	unixListener, err := net.Listen("unix", edge.UnixSock)
	if err != nil {
		return fmt.Errorf("edgeServer: listen unix %w", err)
	}
	log.Infof("Edgekv EdgeServer listen on %s", edge.UnixSock)
	return serve.Serve(unixListener)
}

func (serve *EdgeServer) lastChange(changes diff.Changelog) diff.Change {
	l := len(changes)
	return changes[l-1]
}

func (serve *EdgeServer) watch(prefix string, fn edgekv.ChangeFunc) {
	log.Infof("edge_server: watch '%s'", prefix)
	serve.listener.Watch(prefix, func(key string, payload interface{}) {
		var event = payload.(edgekv.WatchEvent)
		event.Done(fn(event.Key, event.Old, event.Val) == nil)
	})
}

func (serve *EdgeServer) dispatch(key string, event edgekv.WatchEvent) {
	log.Infof("dispatch to '%s'", key)
	serve.listener.Dispatch(key, event)
}

// func (serve *EdgeServer) pushBind(sessID string, key string, timeout time.Duration) {
// 	serve.bindSessions.Store(sessID, msg)
// }

func (serve *EdgeServer) popBind(sessID string) (edgekv.MessageGetBind, bool) {
	if val, ok := serve.bindSessions.LoadAndDelete(sessID); ok {
		return val.(edgekv.MessageGetBind), true
	}

	return edgekv.MessageGetBind{}, false
}

func Start() error {
	return serve.Start()
}

func Stop() error {
	return serve.Stop()
}

func StartUnix(filename string) error {
	edge.UnixSock = filename

	return Start()
}

func SetEdgeID(id edgekv.EdgeID) {
	serve.SetEdgeID(id)
}

func SetStore(store edgekv.Store) {
	serve.SetStore(store)
}

func SetMessageQueue(mq edgekv.MessageQueue) {
	serve.SetMessageQueue(mq)
}

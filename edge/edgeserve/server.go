package edgeserve

import (
	"context"
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

	store    edgekv.Store
	mq       edgekv.MessageQueue
	listener edgekv.Listener
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
	r.HandleFunc("/watch/{key}", serve.Watch).Methods(http.MethodGet)
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

func (serve *EdgeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		key    = strings.TrimPrefix(r.URL.Path, "/")
		q      = r.URL.Query()
		method = r.Method
		err    error
	)

	switch method {
	case http.MethodGet: // Get
		log.Debugf("GET: %s", key)
		if len(q.Get("watch")) > 0 {
			goto Watch
		}
		contentType := r.Header.Get("Content-Type")
		log.Debugf("context type %s", contentType)
		switch contentType {
		case mime.TypeByExtension(".json"), "":
			val, err := serve.getType(key, q)
			if err != nil {
				AbortErr(w, http.StatusNotFound, err)
				return
			}

			log.Debugf("value type %T", val)
			Jsonify(w, &Map{"data": val})
		case mime.TypeByExtension(".gob"):
		}
	case http.MethodPost: // Set
		var (
			val = new(interface{})
			old interface{}
		)

		if err = serve.decodeType(val, readBody(r.Body), q); err != nil {
			AbortErr(w, http.StatusBadRequest, err)
			return
		}

		log.Debugf("POST: %s with value %v", key, *val)
		if old, err = serve.store.Set(key, *val); err != nil {
			AbortErr(w, http.StatusInternalServerError, err)
			return
		}

		if err = serve.Sync(old, *val, key); err != nil {
			AbortErr(w, http.StatusInternalServerError, err)
			return
		}

		Jsonify(w, nil)
	}

	return
Watch:
	type change struct {
		Key    string
		Change interface{}
	}
	var chEvent = make(chan change)

	serve.watch(key, func(key string, old, new interface{}) error {
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
		fmt.Fprintf(w, "change: %s\n\n", utils.Stringify(map[string]interface{}{"key": event.Key, "change": event.Change}))
		f.Flush()
	}
}

// GetKey 取键值
func (serve *EdgeServer) GetKey(w http.ResponseWriter, r *http.Request) {
	var (
		key     = strings.TrimPrefix(r.URL.Path, "/")
		encoder func(val interface{}, q url.Values) []byte
		q       = r.URL.Query()
		val     interface{}
		ok      bool
	)

	log.Debugf("GET: %s", key)

	contentType := r.Header.Get("Content-Type")
	log.Debugf("context type %s", contentType)
	switch contentType {
	case mime.TypeByExtension(".json"), "":
		encoder = serve.encodeJson
	case mime.TypeByExtension(".gob"):
		encoder = serve.encodeGob
	}

	if val, ok = serve.store.Get(key); !ok {
		AbortErr(w, http.StatusNotFound, fmt.Errorf("not found key '%s'", key))
		return
	}

	b := encoder(val, q)
	w.WriteHeader(200)
	w.Header().Set("Content-Type", contentType)
	w.Write(b)
}

// SetKey 设置键值
func (serve *EdgeServer) SetKey(w http.ResponseWriter, r *http.Request) {
	var (
		key      = strings.TrimPrefix(r.URL.Path, "/")
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
	case mime.TypeByExtension(".gob"):
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

func (sever *EdgeServer) Watch(w http.ResponseWriter, r *http.Request) {

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
	return []byte(utils.Stringify(Map{"data": val}))
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
		var val = new(interface{})
		serve.decodeType(val, b, q)
		return *val
	}

	return utils.JSON(string(b))
}

func (serve *EdgeServer) decodeGob(b []byte, q url.Values) interface{} {
	var val = new(interface{})
	if err := utils.Unmarshal(b, val); err == nil {
		return *val
	}
	return nil
}

func (serve *EdgeServer) getType(key string, q url.Values) (val interface{}, err error) {
	var (
		typ = q.Get("type")
		ok  bool
	)

	switch typ {
	case "int":
		v := serve.store.GetInt(key)
		return v, nil
	case "int32":
		v := serve.store.GetInt32(key)
		return v, nil
	case "int64":
		v := serve.store.GetInt64(key)
		return v, nil
	case "uint":
		v := serve.store.GetUint(key)
		return v, nil
	case "uint32":
		v := serve.store.GetUint32(key)
		return v, nil
	case "uint64":
		v := serve.store.GetUint64(key)
		return v, nil
	case "bool":
		v := serve.store.GetBool(key)
		return v, nil
	case "float":
		v := serve.store.GetFloat64(key)
		return v, nil
	case "string":
		v := serve.store.GetString(key)
		return v, nil
	case "duration":
		v := serve.store.GetDuration(key)
		return v, nil
	case "time":
		v := serve.store.GetTime(key)
		return v, nil
	case "[]int":
		v := serve.store.GetIntSlice(key)
		return v, nil
	case "[]string":
		v := serve.store.GetStringSlice(key)
		return v, nil
	case "map":
		v := serve.store.GetStringMap(key)
		return v, nil
	default:
		val, ok = serve.store.Get(key)
		if !ok {
			return nil, fmt.Errorf("not found key %s", key)
		}
		return
	}
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

// func (serve *EdgeServer) Topic()

func Start() error {
	return serve.Start()
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

func init() {
	mime.AddExtensionType(".gob", "application/gob")
}

package edgeserve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/edge"
	"github.com/hysios/log"
	. "github.com/hysios/utils/response"
)

type Map = map[string]interface{}

type EdgeServer struct {
	http.Server
	ID edgekv.EdgeID

	store edgekv.Store
	mq    edgekv.MessageQueue
}

var serve = EdgeServer{}

// Start 启动 Edge 的服务器，服务器主要以下功能
// 1. 管理与 Center 数据同步，消息推送
// 2. 提供 socket 客户端调用 API
func (serve *EdgeServer) Start() error {
	var err error
	serve.Handler = serve
	if serve.mq == nil || serve.store == nil {
		return errors.New("edge_server: mq or store is missing")
	}

	if len(serve.ID) == 0 {
		return errors.New("missing EdgeID")
	}

	if err = serve.mq.Subscribe(serve.ID.Topic(), nil); err != nil {
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
		val, err := serve.getType(key, q)
		if err != nil {
			AbortErr(w, http.StatusNotFound, err)
			return
		}

		log.Debugf("value type %T", val)
		Jsonify(w, &Map{"data": val})
	case http.MethodPost: // Set
		var (
			val = new(interface{})
		)

		if err = serve.decodeType(val, readBody(r.Body), q); err != nil {
			AbortErr(w, http.StatusBadRequest, err)
			return
		}

		log.Debugf("POST: %s with value %v", key, *val)
		serve.store.Set(key, *val)
		Jsonify(w, nil)
	}
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

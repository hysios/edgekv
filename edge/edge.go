package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/hysios/edgekv"
)

var UnixSock = "/var/run/edgekv.sock"

type EdgeStore struct {
	edgekv.Accessor

	client http.Client
	q      url.Values
	_host  string
}

func Open() (edgekv.Store, error) {
	var store = &EdgeStore{}
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", UnixSock)
			},
		},
	}

	store.Accessor = edgekv.MakeAccessor(store)
	store.client = client

	return store, nil
}

func (edge *EdgeStore) SetHost(host string) {
	edge._host = host
}

func (edge *EdgeStore) host(path string) string {
	name := filepath.Base(UnixSock)
	return fmt.Sprintf("http://%s/%s", name, path)
}

type EdgeData struct {
	Data   interface{} `json:"data,omitempty"`
	Status string      `json:"status,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func (edge *EdgeStore) Get(key string) (interface{}, bool) {
	var (
		q    = edge.getQuery()
		path = edge.host(key)
		data EdgeData
	)

	resp, err := edge.get(path, q)
	if err != nil {
		return nil, false
	}

	var dec = json.NewDecoder(resp.Body)

	if err = dec.Decode(&data); err != nil {
		return nil, false
	}

	return data.Data, true
}

func (edge *EdgeStore) getQuery() url.Values {
	return edge.q
}

func (edge *EdgeStore) get(key string, q url.Values) (*http.Response, error) {
	u, err := url.Parse(key)
	if err != nil {
		return nil, err
	}
	u.RawQuery = q.Encode()

	return edge.client.Get(u.String())
}

// func (edge *EdgeStore) getValue(key string, q url.Values) (interface{}, error) {
// 	var (
// 		resp, err = edge.get(key, q)
// 		dec       = json.NewDecoder(resp.Body)
// 		val       = new(interface{})
// 		typ       = q.Get("type")
// 		ok        bool
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	if err = dec.Decode(val); err != nil {
// 		return nil, err
// 	}

// 	switch typ {
// 	case "int":
// 		v := edge.GetInt(key)
// 		return v, nil
// 	case "int32":
// 		v := edge.GetInt32(key)
// 		return v, nil
// 	case "int64":
// 		v := edge.GetInt64(key)
// 		return v, nil
// 	case "uint":
// 		v := edge.GetUint(key)
// 		return v, nil
// 	case "uint32":
// 		v := edge.GetUint32(key)
// 		return v, nil
// 	case "uint64":
// 		v := edge.GetUint64(key)
// 		return v, nil
// 	case "bool":
// 		v := edge.GetBool(key)
// 		return v, nil
// 	case "float":
// 		v := edge.GetFloat64(key)
// 		return v, nil
// 	case "string":
// 		v := edge.GetString(key)
// 		return v, nil
// 	case "duration":
// 		v := edge.GetDuration(key)
// 		return v, nil
// 	case "time":
// 		v := edge.GetTime(key)
// 		return v, nil
// 	case "[]int":
// 		v := edge.GetIntSlice(key)
// 		return v, nil
// 	case "[]string":
// 		v := edge.GetStringSlice(key)
// 		return v, nil
// 	case "map":
// 		v := edge.GetStringMap(key)
// 		return v, nil
// 	default:
// 		val, ok = edge.Get(key)
// 		if !ok {
// 			return nil, fmt.Errorf("not found key %s", key)
// 		}
// 		return
// 	}

// }

func (edge *EdgeStore) Set(key string, val interface{}) {
	panic("not implemented") // TODO: Implement
}

func (edge *EdgeStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (edge *EdgeStore) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

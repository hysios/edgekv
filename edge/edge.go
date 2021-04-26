package edge

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
)

var UnixSock = "/var/run/edgekv.sock"

type EdgeStore struct {
	edgekv.Accessor

	client http.Client
	q      url.Values
	_host  string
}

func Open() (edgekv.Database, error) {
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

type EdgeEvent struct {
	Method    string
	SessionID string
	Key       string
	Change    interface{}
}

func (edge *EdgeStore) Get(key string) (interface{}, bool) {
	var (
		path                                      = edge.host(path.Join("key", key))
		decoder func([]byte) (interface{}, error) = edge.decodeGob
		val     interface{}
	)

	resp, err := edge.get(path)
	if err != nil {
		return nil, false
	}

	b := edge.readBody(resp)
	if val, err = decoder(b); err != nil {
		log.Infof("decoder error %s", err)
		return nil, false
	}

	return val, true
}

func (edge *EdgeStore) Set(key string, val interface{}) {
	var (
		path = edge.host(path.Join("key", key))
		u    *url.URL
		req  *http.Request
		err  error
	)
	if u, err = url.Parse(path); err != nil {
		log.Debugf("parse key '%s' error %s", path, err)
		return
	}

	b := bytes.NewBuffer(edge.encodeGob(val))
	if req, err = http.NewRequest(http.MethodPost, u.String(), b); err != nil {
		log.Debugf("new req error %s", err)
		return
	}
	req.Header.Add("Content-Type", edgekv.BinaryMimeType)
	edge.client.Do(req)
}

func (edge *EdgeStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	var (
		path = edge.host(path.Join("watch", prefix))
		u    *url.URL
		req  *http.Request
		err  error
	)

	if u, err = url.Parse(path); err != nil {
		log.Debugf("parse key '%s' error %s", path, err)
		return
	}

	if req, err = http.NewRequest(http.MethodGet, u.String(), nil); err != nil {
		log.Debugf("new req error %s", err)
		return
	}

	req.Header.Add("Content-Type", edgekv.BinaryMimeType)
	resp, err := edge.client.Do(req)
	if err != nil {
		log.Debugf("req error %s", err)
		return
	}

	s := NewFrameScanner(resp.Body)
	for event := range s.DecodeFrame() {
		fn(event.Key, nil, event.Change)
	}
}

func (edge *EdgeStore) Bind(key string, fn edgekv.BindHandler) error {
	var (
		req  *http.Request
		resp *http.Response
		err  error
		path = edge.parseKey(path.Join("bind_observer", key))
	)

	if req, err = http.NewRequest(http.MethodGet, path, nil); err != nil {
		return fmt.Errorf("new req error %w", err)
	}

	req.Header.Add("Content-Type", edgekv.BinaryMimeType)
	if resp, err = edge.client.Do(req); err != nil {
		return fmt.Errorf("req error %s", err)
	}

	s := NewFrameScanner(resp.Body)
	for event := range s.DecodeFrame() {
		meth := edgekv.BindMethod(event.Method)
		switch meth {
		case edgekv.BindGet:
			readVal, ok := fn(meth, key, nil)
			edge.upstreamSync(event.SessionID, readVal, ok)
		case edgekv.BindSet:
			fn(meth, key, event.Change)
		case edgekv.BindDelete:
			// TODO: Bind Delete
		}
	}
	return nil
}

func (edge *EdgeStore) upstreamSync(sessID string, val interface{}, ok bool) {
	var (
		req  *http.Request
		err  error
		path = edge.parseKey(path.Join("bind", sessID))
	)

	b := edge.encodeGob(val)
	if req, err = http.NewRequest(http.MethodPut, path, bytes.NewBuffer(b)); err != nil {
		log.Debugf("new req error %s", err)
		return
	}

	req.Header.Add("Content-Type", edgekv.BinaryMimeType)
	if _, err = edge.client.Do(req); err != nil {
		log.Debugf("req error %s", err)
		return
	}
}

func (edge *EdgeStore) decodeFrame(frame string) (*EdgeEvent, error) {
	var (
		ss    = strings.Split(frame, "change:")
		event = &EdgeEvent{}
		b     []byte
		err   error
	)

	raw := strings.TrimSpace(ss[1])
	log.Infof("ss: %v", raw)
	if b, err = base64.StdEncoding.DecodeString(raw); err != nil {
		return nil, fmt.Errorf("edge: decoder base64 error %s", err)
	}

	if err = utils.Unmarshal(b, event); err != nil {
		return nil, fmt.Errorf("edge: unmarshal error %s", err)
	}

	return event, nil
}

func (edge *EdgeStore) parseKey(key string) string {
	var (
		path = edge.host(key)
		u    *url.URL
		err  error
	)

	if u, err = url.Parse(path); err != nil {
		log.Debugf("parse key '%s' error %s", path, err)
		return ""
	}

	return u.String()
}

func (edge *EdgeStore) watchSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	p := bytes.IndexAny(data, "\n\n")
	if p > 0 {
		return p, data[:p], nil
	}

	return 0, nil, nil
}

func (edge *EdgeStore) readBody(resp *http.Response) []byte {
	b, _ := ioutil.ReadAll(resp.Body)
	return b
}

func (edge *EdgeStore) decodeJSON(b []byte) (interface{}, error) {
	panic("nonimplement")
}

func (edge *EdgeStore) decodeGob(b []byte) (interface{}, error) {
	var data EdgeData
	if err := utils.Unmarshal(b, &data); err != nil {
		return nil, err
	} else {
		return data.Data, nil
	}
}

func (edge *EdgeStore) encodeGob(val interface{}) []byte {
	var data = EdgeData{Status: "success", Data: val}
	b, _ := utils.Marshal(data)
	return b
}

func (edge *EdgeStore) get(key string) (*http.Response, error) {
	var (
		u   *url.URL
		req *http.Request
		err error
	)

	if u, err = url.Parse(key); err != nil {
		return nil, err
	}

	if req, err = http.NewRequest(http.MethodGet, u.String(), nil); err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", edgekv.BinaryMimeType)

	return edge.client.Do(req)
}

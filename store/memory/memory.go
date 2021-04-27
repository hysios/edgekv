package memory

import (
	"github.com/hysios/edgekv"
	"github.com/hysios/mapindex"
)

type memStore struct {
	values map[string]interface{}
	edgekv.Accessor
}

func OpenMapStore() edgekv.Store {
	store := &memStore{
		values: make(map[string]interface{}),
	}
	store.Accessor = edgekv.MakeAccessor(store)

	return store
}

func (m *memStore) init() {
	if m.values == nil {
		m.values = make(map[string]interface{})
	}
}

func (m *memStore) Get(key string) (val interface{}, ok bool) {
	m.init()

	if val = mapindex.Get(&m.values, key); val != nil {
		ok = true
	}

	return
}

func (m *memStore) Set(key string, val interface{}) (old interface{}, err error) {
	m.init()

	old = mapindex.Get(&m.values, key)
	mapindex.Set(&m.values, key, val, mapindex.OptOverwrite())
	return old, nil
}

func (m *memStore) SetSyncer(_ edgekv.MessageQueue) {
}

func init() {
	edgekv.RegisterStore("memory", func(args ...string) (edgekv.Store, error) {
		return OpenMapStore(), nil
	})
}

var _ edgekv.Store = &memStore{}

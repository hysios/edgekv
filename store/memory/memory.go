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

func (m *memStore) Set(key string, val interface{}) {
	m.init()

	mapindex.Set(&m.values, key, val, mapindex.OptOverwrite())
}

func (m *memStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (m *memStore) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

func init() {
	edgekv.RegisterStore("memory", func(args ...string) (edgekv.Store, error) {
		return OpenMapStore(), nil
	})
}

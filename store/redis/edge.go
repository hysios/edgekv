package redis

import "github.com/hysios/edgekv"

type EdgeStore struct {
	edgekv.Accessor

	ID     edgekv.EdgeID
	master *RedisStore
}

func (redis *RedisStore) OpenEdge(edgeID edgekv.EdgeID) edgekv.Store {
	var store = &EdgeStore{master: redis, ID: edgeID}
	store.Accessor = edgekv.MakeAccessor(store)

	return store
}

func (edge *EdgeStore) Get(key string) (val interface{}, ok bool) {
	var fullkey = edge.master.edgeNode(edge.ID, key)
	return edge.master.Get(fullkey)
}

func (edge *EdgeStore) Keys() []string {
	var (
		fullkey = edge.master.edgeNode(edge.ID, "")
		keys    = edge.master.ListKeys(fullkey)
	)

	return keys
}

func (edge *EdgeStore) Set(key string, val interface{}) (old interface{}, err error) {
	var (
		fullkey = edge.master.edgeNode(edge.ID, key)
	)

	old, _ = edge.Get(key)
	edge.master.Set(fullkey, val)
	return old, nil
}

func (edge *EdgeStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (edge *EdgeStore) Bind(prefix string, fn edgekv.BindHandler) error {
	panic("not implemented") // TODO: Implement
}

var _ edgekv.Store = &EdgeStore{}

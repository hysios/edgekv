package center

import (
	"github.com/hysios/edgekv"
)

type CenterDatabase struct {
	edgekv.Accessor
	ID     edgekv.EdgeID
	master *CenterServer
	store  edgekv.Store
}

func (serve *CenterServer) OpenEdge(edgeID edgekv.EdgeID) edgekv.Database {
	var database = &CenterDatabase{
		ID:     edgeID,
		store:  serve.store.OpenEdge(edgeID),
		master: serve,
	}

	database.Accessor = edgekv.MakeAccessor(database)

	return database
}

func (center *CenterDatabase) Get(key string) (val interface{}, ok bool) {
	return center.store.Get(center.Fullkey(key))
}

func (center *CenterDatabase) Set(key string, val interface{}) {
	center.store.Set(center.Fullkey(key), val)
}

func (center *CenterDatabase) Watch(prefix string, fn edgekv.ChangeFunc) {
	center.master.watch(center.Fullkey(prefix), fn)
}

func (center *CenterDatabase) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

func (center *CenterDatabase) Fullkey(key string) string {
	return center.master.store.EdgeKey(center.ID, key)
}

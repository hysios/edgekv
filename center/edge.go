package center

import (
	"github.com/hysios/edgekv"
)

type CenterDatabase struct {
	edgekv.Accessor
	ID    edgekv.EdgeID
	store edgekv.Store
}

func (serve *CenterServer) OpenEdge(edgeID edgekv.EdgeID) edgekv.Database {
	var database = &CenterDatabase{
		ID:    edgeID,
		store: serve.store.OpenEdge(edgeID),
	}

	database.Accessor = edgekv.MakeAccessor(database)

	return database
}

func (center *CenterDatabase) Get(key string) (val interface{}, ok bool) {
	panic("not implemented") // TODO: Implement
}

func (center *CenterDatabase) Set(key string, val interface{}) {
	panic("not implemented") // TODO: Implement
}

func (center *CenterDatabase) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (center *CenterDatabase) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

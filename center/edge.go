package center

import (
	"reflect"

	"github.com/hysios/edgekv"
	"github.com/hysios/log"
	"github.com/r3labs/diff/v2"
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

	database.Accessor = edgekv.MakeAccessor(&CenterWrap{database})

	return database
}

type CenterWrap struct {
	*CenterDatabase
}

func (wrap *CenterWrap) Get(key string) (val interface{}, ok bool) {
	return wrap.CenterDatabase.Get(key)
}

func (wrap *CenterWrap) Keys() []string {
	//TODO:
	panic("nonimplement")

}

func (center *CenterDatabase) Get(key string, opts ...edgekv.GetOpt) (val interface{}, ok bool) {
	return center.store.Get(key)
}

func (center *CenterDatabase) Set(key string, val interface{}, opts ...edgekv.SetOpt) {
	var (
		old interface{}
		err error
	)

	if old, err = center.store.Set(key, val); err != nil {
		log.Errorf("center_database: set '%s' error: %s", center.Fullkey(key), err)
		return
	}

	if err = center.Sync(old, val, key); err != nil {
		log.Errorf("center_database: set '%s' error: %s", center.Fullkey(key), err)
		return
	}
}

func (center *CenterDatabase) Sync(old, val interface{}, key string) error {
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

	topic := center.Fullkey("sync")
	return center.master.mq.Publish(topic, edgekv.Message{
		From: string(center.ID),
		Type: edgekv.CmdChangelog,
		Payload: edgekv.MessageChangelog{
			Key:     key,
			Changes: changes,
		},
	})
}

func (center *CenterDatabase) Watch(pattern string, fn edgekv.ChangeFunc) {
	center.master.watch(center.Fullkey(pattern), fn)
}

func (center *CenterDatabase) Bind(key string, fn edgekv.BindHandler) error {
	panic("not implemented") // TODO: Implement
}

func (center *CenterDatabase) Fullkey(key string) string {
	return center.master.store.EdgeKey(center.ID, key)
}

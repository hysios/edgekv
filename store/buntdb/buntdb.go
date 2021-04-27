package buntdb

import (
	"errors"
	"fmt"

	"github.com/fatih/structs"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
	"github.com/hysios/mapindex"
	"github.com/tidwall/buntdb"
)

type buntdbStore struct {
	edgekv.Accessor

	ID edgekv.EdgeID
	db *buntdb.DB
}

func OpenBuntDBStore(filename string) (*buntdbStore, error) {
	var (
		store = &buntdbStore{}
		db    *buntdb.DB
		err   error
	)

	if db, err = buntdb.Open(filename); err != nil {
		return nil, err
	}

	store.Accessor = edgekv.MakeAccessor(store)
	store.db = db
	return store, nil
}

func (store *buntdbStore) Get(key string) (val interface{}, ok bool) {
	var err error
	if val, err = store.get(key); err != nil {
		log.Infof("buntdb: get key '%s' error: %s", key, err)
		return nil, false
	}
	return val, true
}

func (store *buntdbStore) Set(key string, val interface{}) (old interface{}, err error) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		m              = make(map[string]interface{})
	)

	store.db.View(func(tx *buntdb.Tx) error {
		raw, err := tx.Get(prefix)
		if err != nil {
			return err
		}

		if err = utils.Unmarshal([]byte(raw), &m); err != nil {
			return fmt.Errorf("buntdb_store: unmarshal error: %w", err)
		}

		return nil
	})

	if len(subkey) > 0 {
		old = mapindex.Get(m, subkey)
		mapindex.Set(&m, subkey, val, mapindex.OptOverwrite())
	} else {
		old = m

		switch x := val.(type) {
		case map[string]interface{}:
			m = x
		default:
			m = structs.Map(val)
		}

	}

	if err = store.db.Update(func(tx *buntdb.Tx) error {
		var b []byte
		// _, _, err = tx.Set(prefix,  utils.Stringify(m), nil)
		if b, err = utils.Marshal(m); err != nil {
			return err
		}
		_, _, err = tx.Set(prefix, string(b), nil)
		return err
	}); err != nil {
		return nil, err
	}

	return

}

func (store *buntdbStore) get(key string) (val interface{}, err error) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		raw            string
		out            = make(map[string]interface{})
	)

	store.db.View(func(tx *buntdb.Tx) error {
		if raw, err = tx.Get(prefix); err != nil {
			return err
		}
		return utils.Unmarshal([]byte(raw), &out)
	})

	if len(subkey) > 0 {
		val = mapindex.Get(out, subkey)
	} else {
		val = out
	}
	return val, nil
}

type (
	Finder func(fn func(tx *buntdb.Tx) error) error
	OpFunc func(prefix, subkey, raw string, tx *buntdb.Tx) error
)

func (store *buntdbStore) stepSelect(key string, fn Finder, cb OpFunc) (err error) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		raw            string
	)

	if err = fn(func(tx *buntdb.Tx) error {
		if raw, err = tx.Get(key); errors.Is(err, buntdb.ErrNotFound) {
			if raw, err = tx.Get(prefix); err != nil {
				return err
			}
		} else {
			subkey = ""
		}
		return cb(prefix, subkey, raw, tx)
	}); err != nil {
		return err
	}
	return nil
}

// func (store *buntdbStore) Sync(changes diff.Changelog) error {
// 	log.Debugf("sync diff %#v", changes)
// 	// store.mq.Publish()
// 	return errors.New("nonimplement")
// }

func init() {
	edgekv.RegisterStore("buntdb", func(args ...string) (edgekv.Store, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("missing open filename of buntdb")
		}

		if store, err := OpenBuntDBStore(args[0]); err != nil {
			return nil, fmt.Errorf("buntdb_store: open buntdb error %w", err)
		} else {
			return store, nil
		}
	})
}

var _ edgekv.Store = &buntdbStore{}

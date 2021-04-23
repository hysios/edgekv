package buntdb

import (
	"errors"
	"fmt"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
	"github.com/hysios/mapindex"
	"github.com/r3labs/diff"
	"github.com/tidwall/buntdb"
)

type buntdbStore struct {
	edgekv.Accessor

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

func (store *buntdbStore) Set(key string, val interface{}) {
	var (
		prefix, subkey string
		parent         interface{}
		keyval         interface{}
		err            error
		changes        diff.Changelog
	)

	// 阶段载入 key 的父对像与子键值
	if err = store.stepSelect(key, store.db.View, func(_prefix, _subkey, raw string, tx *buntdb.Tx) error {
		prefix = _prefix
		subkey = _subkey
		parent = utils.JSON(raw)
		if len(_subkey) > 0 {
			keyval = mapindex.Get(parent, subkey)
		}
		return nil
	}); err != nil {
		log.Infof("buntdb: set key %s with val %v error: %s", key, val, err)
	}

	if len(subkey) > 0 { // 子值存储
		if err = mapindex.Set(parent, subkey, val); err != nil {
			log.Error(err)
			return
		}
		// 存储到指定的 key 数据库中去
		if err = store.db.Update(func(tx *buntdb.Tx) error {
			_, _, err = tx.Set(prefix, utils.Stringify(parent), nil)
			return err
		}); err != nil {
			log.Errorf("buntdb: set new value error: %s", err)
			return
		}
		changes = diff.Changelog{{Type: diff.UPDATE, Path: []string{key}, From: keyval, To: val}}
	} else {
		// 存储整个map数据库中去
		if err = store.db.Update(func(tx *buntdb.Tx) error {
			_, _, err = tx.Set(prefix, utils.Stringify(val), nil)
			return err
		}); err != nil {
			log.Errorf("buntdb: set new value error: %s", err)
			return
		}

		if changes, err = diff.Diff(parent, val); err != nil {
			changes = diff.Changelog{{
				Type: diff.DELETE, Path: []string{key},
			}, {
				Type: diff.CREATE, Path: []string{key}, To: val,
			}}
		}
	}

	store.Sync(changes)
}

func (store *buntdbStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *buntdbStore) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *buntdbStore) get(key string) (val interface{}, err error) {

	var subkey string
	if err = store.stepSelect(key, store.db.View, func(prefix, _subkey, raw string, tx *buntdb.Tx) error {
		val = utils.JSON(raw)
		subkey = _subkey

		return nil
	}); err != nil {
		return nil, err
	}

	switch x := val.(type) {
	case map[string]interface{}:
		if len(subkey) > 0 {
			val = mapindex.Get(x, subkey)
		}
		return val, nil

	default:
		return x, nil
	}
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

func (store *buntdbStore) Sync(changes diff.Changelog) error {
	log.Debugf("sync diff %#v", changes)
	return errors.New("nonimplement")
}

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

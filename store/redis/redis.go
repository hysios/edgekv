package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
	"github.com/hysios/mapindex"
)

type EdgeNoder func(edgeID edgekv.EdgeID, suffix string) string

type RedisStore struct {
	edgekv.Accessor
	DB     int
	Prefix string

	rdb      *redis.Client
	edgeNode EdgeNoder
}

func OpenRedisStore(uri string) (*RedisStore, error) {
	var (
		store = &RedisStore{}
		rdb   *redis.Client
		err   error
	)

	if rdb, err = store.ParseURI(uri); err != nil {
		return nil, err
	}

	store.Accessor = edgekv.MakeAccessor(store)
	store.rdb = rdb
	store.edgeNode = store.edgekey

	return store, nil
}

func (store *RedisStore) ParseURI(uri string) (*redis.Client, error) {
	var u, err = url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("redis_store: parse open uri error %w", err)
	}

	opts := &redis.Options{}

	if pass, ok := u.User.Password(); ok {
		opts.Password = pass
	}

	store.parseQuery(opts, u.Query())
	rdb := redis.NewClient(opts)

	return rdb, nil
}

func (store *RedisStore) parseQuery(opts *redis.Options, q url.Values) {
	for key := range q {
		switch key {
		case "db":
			db, err := strconv.Atoi(q.Get(key))
			if err == nil {
				opts.DB = db
			}
		}
	}
}

func (store *RedisStore) Get(key string) (val interface{}, ok bool) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		m              = make(map[string]interface{})
		err            error
		raw            []byte
	)

	if raw, err = store.rdb.Get(prefix).Bytes(); err != nil {
		log.Debugf("redis_store: get key %s error: %s", prefix, err)
		return nil, false
	}

	if err = json.Unmarshal(raw, &m); err != nil {
		log.Debugf("redis_store: unmarshal json error: %s", err)
		return nil, false
	}

	val = mapindex.Get(m, subkey)
	return val, val != nil
}

func (store *RedisStore) edgekey(edgeID edgekv.EdgeID, key string) string {
	return fmt.Sprintf("%s:%s:%s", store.Prefix, edgeID, key)
}

func (store *RedisStore) Set(key string, val interface{}) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		m              = make(map[string]interface{})
		err            error
		raw            []byte
	)

	if raw, err = store.rdb.Get(prefix).Bytes(); err == nil {
		// log.Debugf("redis_store: get key %s error: %s", prefix, err)

		if err = json.Unmarshal(raw, &m); err != nil {
			log.Debugf("redis_store: unmarshal json error: %s", err)
			return
		}
	}

	if len(subkey) > 0 {
		mapindex.Set(&m, subkey, mapindex.OptOverwrite())
	} else {
		switch x := val.(type) {
		case map[string]interface{}:
			m = x
		default:
			m = structs.Map(x)
		}
	}

	if _, err := store.rdb.Set(prefix, utils.Stringify(m), -1).Result(); err != nil {
		log.Debugf("redis_store: set key %s error: %s", prefix, err)
	}
}

func (store *RedisStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *RedisStore) WatchEdges(prefix string, fn edgekv.EdgeChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *RedisStore) Bind(prefix string, fn edgekv.ReaderFunc) {
	panic("not implemented") // TODO: Implement
}

func init() {
	edgekv.RegisterStore("redis", func(args ...string) (edgekv.Store, error) {
		if len(args) < 1 {
			return nil, errors.New("redis_store: missing open uri")
		}

		return OpenRedisStore(args[0])
	})
}

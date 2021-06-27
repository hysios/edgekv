package redis

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/go-redis/redis/v8"
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
	store.edgeNode = store.EdgeKey

	return store, nil
}

func (store *RedisStore) fullkey(key string) string {
	return store.Prefix + ":" + key
}

func (store *RedisStore) ParseURI(uri string) (*redis.Client, error) {
	log.Infof("open redis store at %s", uri)
	var u, err = url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("redis_store: parse open uri error %w", err)
	}

	opts := &redis.Options{}

	if pass, ok := u.User.Password(); ok {
		opts.Password = pass
	}

	store.Prefix = strings.TrimPrefix(u.Path, "/")
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
		ctx            = context.Background()
	)

	if raw, err = store.rdb.Get(ctx, store.fullkey(prefix)).Bytes(); err != nil {
		log.Debugf("redis_store: get key '%s' error: %s", prefix, err)
		return nil, false
	}

	if err = utils.Unmarshal(raw, &m); err != nil {
		log.Debugf("redis_store: unmarshal  error: %s", err)
		return nil, false
	}

	if len(subkey) > 0 {
		val = mapindex.Get(m, subkey)
	} else {
		val = m
	}
	return val, val != nil
}

func (store *RedisStore) ListKeys(prefix string) []string {
	var ctx = context.Background()
	keys, _, _ := store.rdb.Scan(ctx, 0, store.fullkey(prefix+"*"), -1).Result()
	for i, key := range keys {
		keys[i] = strings.TrimPrefix(key, store.fullkey(prefix))
	}
	return keys
}

func (store *RedisStore) Keys() []string {
	//TODO: 实现 keys
	panic("nonimplement")
}

func (store *RedisStore) EdgeKey(edgeID edgekv.EdgeID, key string) string {
	return edgekv.Edgekey(edgeID, key)
}

var timeType = reflect.TypeOf(new(time.Time)).Elem()

func (store *RedisStore) value(val interface{}) interface{} {
	v := reflect.ValueOf(val)
	v = reflect.Indirect(v)
	switch v.Kind() {
	case reflect.Map:
		return val
	case reflect.Struct:
		if v.Type() != timeType {
			return structs.Map(val)
		} else {
			return val
		}
	default:
		return val
	}
}

func (store *RedisStore) Set(key string, val interface{}) (old interface{}, err error) {
	var (
		prefix, subkey = edgekv.SplitKey(key)
		m              = make(map[string]interface{})
		raw            []byte
		ctx            = context.Background()
	)

	val = store.value(val)

	if raw, err = store.rdb.Get(ctx, store.fullkey(prefix)).Bytes(); err != nil {
		log.Debugf("redis_store: get key '%s' error: %s", prefix, err)
	}

	if len(raw) > 0 {
		if err = utils.Unmarshal(raw, &m); err != nil {
			log.Debugf("redis_store: unmarshal error: %s", err)
			return
		}
	}

	if len(subkey) > 0 {
		old = mapindex.Get(m, subkey)
		mapindex.Set(&m, subkey, val, mapindex.OptOverwrite())
	} else {
		old = m
		switch x := val.(type) {
		case map[string]interface{}:
			m = x
		default:
			m = structs.Map(x)
		}
	}

	var b []byte
	if b, err = utils.Marshal(m); err != nil {
		return nil, err
	}

	if _, err := store.rdb.Set(ctx, store.fullkey(prefix), string(b), -1).Result(); err != nil {
		// if _, err := store.rdb.Set(prefix, utils.Stringify(m), -1).Result(); err != nil {
		log.Debugf("redis_store: set key %s error: %s", prefix, err)
	}

	return
}

func (store *RedisStore) Watch(prefix string, fn edgekv.ChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *RedisStore) WatchEdges(prefix string, fn edgekv.EdgeChangeFunc) {
	panic("not implemented") // TODO: Implement
}

func (store *RedisStore) Bind(prefix string, fn edgekv.BindHandler) error {
	panic("not implemented") // TODO: Implement
}

func (store *RedisStore) SetSyncer(_ edgekv.MessageQueue) {
}

func init() {
	edgekv.RegisterStore("redis", func(args ...string) (edgekv.Store, error) {
		if len(args) < 1 {
			return nil, errors.New("redis_store: missing open uri")
		}

		return OpenRedisStore(args[0])
	})
}

var _ edgekv.CenterStore = &RedisStore{}

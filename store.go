package edgekv

import (
	"fmt"
	"time"
)

type (
	ChangeFunc     func(key string, changs ChangeLog) error
	EdgeChangeFunc func(key string, edgeID EdgeID, changes ChangeLog) error
	ReaderFunc     func(key string) (interface{}, bool)
)

type Store interface {
	Get(key string) (val interface{}, ok bool)
	Set(key string, val interface{}) (old interface{}, err error)
	Watch(prefix string, fn ChangeFunc)
	Bind(prefix string, fn ReaderFunc)
	Accessor
}

type Publisher interface {
}

type WatchEdge interface {
	WatchEdges(prefix string, fn EdgeChangeFunc)
}

type CenterStore interface {
	Store
	WatchEdge
	OpenEdge(edgeID EdgeID) Store
}

type Accessor interface {
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetInt32(key string) int32
	GetInt64(key string) int64
	GetUint(key string) uint
	GetUint32(key string) uint32
	GetUint64(key string) uint64
	GetFloat64(key string) float64
	GetTime(key string) time.Time
	GetDuration(key string) time.Duration
	GetIntSlice(key string) []int
	GetStringSlice(key string) []string
	GetStringMap(key string) map[string]interface{}
	GetStringMapString(key string) map[string]string
	GetStringMapStringSlice(key string) map[string][]string
	GetSizeInBytes(key string) uint
	// UnmarshalKey(key string, rawVal interface{}, opts ...DecoderConfigOption) error
	// Unmarshal(rawVal interface{}, opts ...DecoderConfigOption) error
	// UnmarshalExact(rawVal interface{}, opts ...DecoderConfigOption) error
	// AllKeys() []string
	// AllSettings() map[string]interface{}
}

type OpenStoreFunc func(args ...string) (Store, error)

var stores = make(map[string]OpenStoreFunc)

func RegisterStore(name string, opener OpenStoreFunc) {
	if _, ok := stores[name]; ok {
		return
	}
	stores[name] = opener
}

func OpenStore(name string, args ...string) (Store, error) {
	if opener, ok := stores[name]; !ok {
		return nil, fmt.Errorf("not found store type %s", name)
	} else {
		return opener(args...)
	}
}

package edgekv

import (
	"time"

	"github.com/hysios/mapindex"
	"github.com/hysios/utils/convert"
)

type Getter interface {
	Get(key string) (interface{}, bool)
	Keys() []string
}

type accessor struct {
	getter   Getter
	defaults map[string]interface{}
}

func (a *accessor) get(key string) interface{} {
	if v, ok := a.getter.Get(key); ok {
		return v
	} else if a.defaults != nil {
		return mapindex.Get(a.defaults, key)
	}

	return nil
}

func (a *accessor) GetString(key string) string {
	if v, ok := convert.String(a.get(key)); ok {
		return v
	}

	return ""
}

func (a *accessor) GetBool(key string) bool {
	if v, ok := convert.Bool(a.get(key)); ok {
		return v
	}

	return false
}

func (a *accessor) GetInt(key string) int {
	if v, ok := convert.Int(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetInt32(key string) int32 {
	if v, ok := convert.Int32(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetInt64(key string) int64 {
	if v, ok := convert.Int64(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetUint(key string) uint {
	if v, ok := convert.Uint(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetUint32(key string) uint32 {
	if v, ok := convert.Uint32(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetUint64(key string) uint64 {
	if v, ok := convert.Uint64(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetFloat64(key string) float64 {
	if v, ok := convert.Float(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetTime(key string) time.Time {
	if v, ok := convert.Time(a.get(key)); ok {
		return v
	}

	return time.Time{}
}

func (a *accessor) GetDuration(key string) time.Duration {
	if v, ok := convert.Duration(a.get(key)); ok {
		return v
	}

	return 0
}

func (a *accessor) GetIntSlice(key string) []int {
	if v, ok := convert.SliceInt(a.get(key)); ok {
		return v
	}

	return nil
}

func (a *accessor) GetStringSlice(key string) []string {
	if v, ok := convert.SliceString(a.get(key)); ok {
		return v
	}

	return nil
}

func (a *accessor) GetStringMap(key string) map[string]interface{} {
	if v, ok := convert.Map(a.get(key)); ok {
		return v
	}

	return nil
}

func (a *accessor) GetStringMapString(key string) map[string]string {
	if v, ok := convert.MapString(a.get(key)); ok {
		return v
	}

	return nil
}

func (a *accessor) GetStringMapStringSlice(key string) map[string][]string {
	panic("not implemented") // TODO: Implement
}

func (a *accessor) GetSizeInBytes(key string) uint {
	panic("not implemented") // TODO: Implement
}

func (a *accessor) SetDefault(key string, val interface{}) {
	if a.defaults == nil {
		a.defaults = make(map[string]interface{})
	}

	mapindex.Set2(a.defaults, key, val)
}

func (a *accessor) AllKeys() []string {
	return a.getter.Keys()
}

func (a *accessor) AllSettings() map[string]interface{} {
	var m = make(map[string]interface{})
	for _, key := range a.getter.Keys() {
		m[key] = a.get(key)
	}

	return m
}

func MakeAccessor(getter Getter) Accessor {
	return &accessor{getter: getter}
}

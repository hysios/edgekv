package edge

import (
	"testing"
	"time"

	"github.com/hysios/edgekv"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	UnixSock = "/tmp/edgekv.sock"
	m.Run()
}

func TestEdgeStore_Get(t *testing.T) {
	store, err := Open()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	v, ok := store.Get("test")
	assert.True(t, ok)
	assert.NotNil(t, v)
	tt := store.GetTime("test.createdAt")
	assert.NotZero(t, tt)
	t.Logf("time %s", tt)
}

func TestEdgeStore_Set(t *testing.T) {
	store, err := Open()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	store.Set("test.on", false)

	v := store.GetBool("test.on")
	assert.False(t, v)
	store.Set("test.updatedAt", time.Date(2020, 10, 4, 01, 02, 03, 04, time.UTC))
	tt := store.GetTime("test.updatedAt")
	assert.Equal(t, tt, time.Date(2020, 10, 4, 01, 02, 03, 04, time.UTC))
	t.Logf("time %s", tt)
}

func TestEdgeStore_Watch(t *testing.T) {
	store, err := Open()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	go store.Watch("test.*", func(key string, old, new interface{}) error {
		return nil
	})
}

func TestEdgeStore_Bind(t *testing.T) {
	store, err := Open()
	assert.NoError(t, err)
	assert.NotNil(t, store)

	store.Bind("test.v", func(mt edgekv.BindMethod, key string, val interface{}) (interface{}, bool) {
		panic("nonimplement")
	})
}

func TestEdgeStore_Keys(t *testing.T) {
	store, err := Open()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	keys := store.AllKeys()
	assert.NotNil(t, keys)
	assert.Greater(t, len(keys), 0)
	t.Logf("keys %v", keys)
}

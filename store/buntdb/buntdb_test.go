package buntdb

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fatih/structs"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/buntdb"
)

type Map = map[string]interface{}

func testServer() edgekv.Store {
	f, _ := ioutil.TempFile("", "test.db")
	f.Close()
	os.Remove(f.Name())

	store, _ := OpenBuntDBStore(f.Name())

	loadData(store, Map{
		"test": Map{
			"id":        1234,
			"createdAt": time.Date(2020, 1, 2, 12, 59, 59, 0, time.UTC),
			"on":        true,
		},
		"user": Map{
			"username": "Bob",
			"sex":      "male",
			"avatar": []string{
				"http://example.com/avatar1.jpg",
				"http://example.com/avatar2.jpg",
			},
			"profile": Map{
				"money": 1234.00,
				"friends": []interface{}{
					Map{"id": 1, "memoname": "Jim"},
					Map{"id": 1},
				},
			},
		},
	})

	return store
}

func loadData(store edgekv.Store, val interface{}) {
	var (
		vals Map
		err  error
	)

	if _v, ok := val.(map[string]interface{}); ok {
		vals = _v
	} else {
		vals = structs.Map(val)
	}

	bunt := store.(*buntdbStore)

	bunt.db.Update(func(tx *buntdb.Tx) error {
		for key, val := range vals {
			var b []byte
			if b, err = utils.Marshal(val); err != nil {
				return err
			}
			// tx.Set(key, utils.Stringify(val), nil)
			tx.Set(key, string(b), nil)
		}
		return nil
	})
	time.Sleep(100 * time.Millisecond)
}

func Test_buntdbStore_Get(t *testing.T) {
	store := testServer()
	v, ok := store.Get("_notfoundkey")
	assert.False(t, ok)
	assert.Nil(t, v)

	b := store.GetBool("test.on")
	assert.True(t, b)

	tt := store.GetTime("test.createdAt")
	assert.Equal(t, tt, time.Date(2020, 1, 2, 12, 59, 59, 0, time.UTC))

	id := store.GetInt("test.id")
	assert.Equal(t, id, 1234)

	username := store.GetString("user.username")
	assert.Equal(t, username, "Bob")

	money := store.GetFloat64("user.profile.money")
	assert.Equal(t, money, 1234.0)

	v, ok = store.Get("test")
	assert.True(t, ok)
	t.Logf("test: %v", v)
}

func Test_buntdbStore_Set(t *testing.T) {
	store := testServer()

	store.Set("test.id", 1235)

	assert.Equal(t, store.GetInt("test.id"), 1235)
	assert.NotEqual(t, store.GetInt("test.id"), 1236)

	m, ok := store.Get("test")
	assert.True(t, ok)
	assert.Equal(t, m, map[string]interface{}{"createdAt": time.Date(2020, 1, 2, 12, 59, 59, 0, time.UTC), "id": 1235, "on": true})

	store.Set("user.profile.money", 10000.00)
	assert.Equal(t, store.GetFloat64("user.profile.money"), 10000.0)

	store.Set("test", map[string]interface{}{
		"name":      "Alice",
		"age":       18,
		"updatedAt": time.Date(2020, 1, 24, 23, 59, 59, 0, time.UTC),
	})

	v, ok := store.Get("test")
	assert.True(t, ok)
	t.Logf("test: %v", v)
}

func Test_buntdbStore_Keys(t *testing.T) {
	store := testServer()

	keys := store.AllKeys()
	assert.NotNil(t, keys)
	assert.Greater(t, len(keys), 0)
	t.Logf("keys %v", keys)
}

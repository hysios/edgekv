package redis

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/fatih/structs"
	"github.com/go-redis/redis/v8"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
)

type testGets struct {
	store edgekv.Store
	t     *testing.T
}

type testDoc struct {
	User testUser `json:"user,omitempty"`
}

type testUser struct {
	Username  string        `json:"username,omitempty"`
	Age       int           `json:"age,omitempty"`
	Off       bool          `json:"off,omitempty"`
	Price     float64       `json:"price,omitempty"`
	CreatedAt time.Time     `json:"createdAt,omitempty"`
	Timeout   time.Duration `json:"timeout,omitempty"`
	Tags      []string      `json:"tags,omitempty"`
	Image     []byte        `json:"image,omitempty"`
	Org       struct {
		OrgUID  string `json:"orgUID,omitempty"`
		Name    string `json:"name,omitempty"`
		Members int    `json:"members,omitempty"`
		Owner   struct {
			Username string `json:"username,omitempty"`
			Email    string `json:"email,omitempty"`
		} `json:"owner,omitempty"`
	} `json:"org,omitempty"`
	Friends []struct {
		Name string `json:"name,omitempty"`
		Sex  int    `json:"sex,omitempty"`
	} `json:"friends,omitempty"`
}

func (tests *testGets) get(key string, want interface{}, wanOk bool) {
	var tt = struct {
		name    string
		key     string
		wantVal interface{}
		wantOk  bool
	}{
		key:     key,
		wantVal: want,
		wantOk:  wanOk,
	}

	tests.t.Run(tt.name, func(t *testing.T) {
		gotVal, gotOk := tests.store.Get(tt.key)
		if !reflect.DeepEqual(gotVal, tt.wantVal) {
			t.Errorf("RedisStore.Get() gotVal = %v, want %v", gotVal, tt.wantVal)
		}
		if gotOk != tt.wantOk {
			t.Errorf("RedisStore.Get() gotOk = %v, want %v", gotOk, tt.wantOk)
		}
	})
}

type logger struct {
}

func (*logger) Printf(ctx context.Context, format string, v ...interface{}) {
	log.Infof(format, v...)
}

func TestMain(m *testing.M) {
	redis.SetLogger(&logger{})
	log.SetLevel(-1)

	m.Run()
}

func testServer(t *testing.T) *RedisStore {
	store, err := OpenRedisStore("redis://127.0.0.1:6379/?db=3")
	if err != nil {
		t.Fatalf("open redis failed %s", err)
	}

	var (
		user = testUser{
			Username:  "小张",
			Age:       18,
			Off:       true,
			Price:     123.45,
			CreatedAt: time.Date(1980, 1, 1, 12, 34, 45, 0, time.UTC),
			Timeout:   34 * time.Second,
			Tags:      []string{"high", "middle" /* "low", */, "nothing"},
			Image:     []byte("JPEGx011\x11\x12"),
			Org: struct {
				OrgUID  string `json:"orgUID,omitempty"`
				Name    string `json:"name,omitempty"`
				Members int    `json:"members,omitempty"`
				Owner   struct {
					Username string `json:"username,omitempty"`
					Email    string `json:"email,omitempty"`
				} `json:"owner,omitempty"`
			}{
				OrgUID:  "12345",
				Name:    "大中华区",
				Members: 10,
				Owner: struct {
					Username string `json:"username,omitempty"`
					Email    string `json:"email,omitempty"`
				}{
					Username: "jim",
					Email:    "jim@example.com",
				},
			},
			Friends: []struct {
				Name string `json:"name,omitempty"`
				Sex  int    `json:"sex,omitempty"`
			}{{
				"Jim",
				1,
			}, {
				"Alice",
				0,
			}},
		}
		doc = testDoc{
			User: user,
		}
	)

	testStore(store, doc)
	return store
}

func testStore(store *RedisStore, val interface{}) {
	var err error
	var v map[string]interface{}
	switch x := val.(type) {
	case *map[string]interface{}:
		v = *x
	default:
		v = structs.Map(x)
	}

	for key, partv := range v {
		var b []byte
		if b, err = utils.Marshal(partv); err != nil {
			log.Errorf("marshal %v", err)
			continue
		}
		store.rdb.Set(key, b, -1).Err()
	}

}

func TestRedisStore_Get(t *testing.T) {
	store := testServer(t)

	var tests = &testGets{store: store, t: t}
	tests.get("User.Username", "小张", true)
	tests.get("User.Age", 18, true)
	tests.get("User.Price", 123.45, true)
	tests.get("User.Org.OrgUID", "12345", true)
	tests.get("User.CreatedAt", time.Date(1980, 1, 1, 12, 34, 45, 0, time.UTC), true)
	// bad test
	tests.get("User.UpdatedAt", nil, false)
}

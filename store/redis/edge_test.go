package redis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEdgeStore_Set(t *testing.T) {

	store, err := OpenRedisStore("redis://127.0.0.1:6379/?db=3")
	if err != nil {
		t.Fatalf("open redis failed %s", err)
	}

	edgestore := store.OpenEdge("ABETEST1")
	edgestore.Set("user", testUser{
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
	})

	name := edgestore.GetString("user.Username")
	assert.Equal(t, name, "小张")
	age := edgestore.GetInt("user.Age")
	assert.Equal(t, age, 18)
	createdAt := edgestore.GetTime("user.CreatedAt")
	assert.Equal(t, createdAt, time.Date(1980, 1, 1, 12, 34, 45, 0, time.UTC))

	edgestore2 := store.OpenEdge("ABETEST2")
	edgestore2.Set("user", testUser{
		Username:  "Bob",
		Age:       23,
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
	})

	name = edgestore2.GetString("user.Username")
	assert.Equal(t, name, "Bob")
	age = edgestore2.GetInt("user.Age")
	assert.Equal(t, age, 23)
	createdAt = edgestore2.GetTime("user.CreatedAt")
	assert.Equal(t, createdAt, time.Date(1980, 1, 1, 12, 34, 45, 0, time.UTC))
}

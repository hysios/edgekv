package edge

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

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

func TestEdgeStore_get(t *testing.T) {
	type fields struct {
		Accessor edgekv.Accessor
		client   http.Client
		q        url.Values
	}
	type args struct {
		key string
		q   url.Values
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *http.Response
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &EdgeStore{
				Accessor: tt.fields.Accessor,
				client:   tt.fields.client,
				q:        tt.fields.q,
			}
			got, err := edge.get(tt.args.key, tt.args.q)
			if (err != nil) != tt.wantErr {
				t.Errorf("EdgeStore.get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EdgeStore.get() = %v, want %v", got, tt.want)
			}
		})
	}
}

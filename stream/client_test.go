package stream

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tj/assert"
)

func TestClient_Connect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			serve *Server
			err   error
		)

		t.Run("stream_server", func(tt *testing.T) {
			serve, err = Upgrade(w, r)
			assert.NoError(t, err)
			assert.NotNil(t, serve)

			tt.Run("message", func(ttt *testing.T) {
				serve.Message(func(mt MessageType, msg []byte) {
					log.Printf("msg: %s", string(msg))
					assert.Equal(t, mt, StBinary)
					assert.Equal(t, msg, []byte("hello world"))
					serve.Send([]byte("reply hello"))
				})
			})
		})

	}))

	defer ts.Close()

	client, err := NewClient(ts.URL)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	client.Send([]byte("hello world"))

	t.Run("receive", func(tt *testing.T) {
		client.Message(func(_ MessageType, msg []byte) {

			log.Printf("receive: %s", string(msg))
			assert.Equal(t, msg, []byte("reply hello"))
		})
	})
	time.Sleep(300 * time.Millisecond)
}

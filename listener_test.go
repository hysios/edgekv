package edgekv

import (
	"testing"
	"time"
)

func wait() {
	time.Sleep(100 * time.Millisecond)
}

func TestListener_Dispatch(t *testing.T) {
	var listen = NewListner()
	defer listen.Close()

	listen.Watch("test.*", func(key string, payload interface{}) {
		t.Logf("key '%s' => %v", key, payload)
	})

	listen.Watch("test.on", func(key string, payload interface{}) {
		t.Logf("on key '%s' => %v", key, payload)
	})

	t.Run("dispatch", func(tt *testing.T) {
		listen.Dispatch("test.on", true)
	})
	t.Run("dispatch", func(tt *testing.T) {
		listen.Dispatch("test", false)
	})
	wait()
}

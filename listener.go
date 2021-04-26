package edgekv

import (
	"context"
	"path/filepath"
	"sync"

	"go.uber.org/atomic"
)

type Matcher interface {
	Match(ctx context.Context, input string) bool
}

type KeyMatch struct {
	Pattern string
}

func (key *KeyMatch) Match(_ context.Context, input string) bool {
	matched, _ := filepath.Match(key.Pattern, input)
	return matched
}

type SubscribeFunc func(key string, payload interface{})
type Subscribe struct {
	Matcher  Matcher
	Callback SubscribeFunc
}

type DispatchEvent struct {
	Key     string
	Payload interface{}
}

type Listener struct {
	subscribes []Subscribe

	run        atomic.Bool
	dispatchCh chan DispatchEvent
	subLock    sync.RWMutex
}

func NewListner() *Listener {
	listen := &Listener{}

	go listen.Start()

	return listen
}

func (listen *Listener) init() {
	listen.dispatchCh = make(chan DispatchEvent)
}

func (listen *Listener) Start() error {
	if listen.run.Load() {
		return nil
	}

	listen.init()

	listen.run.Store(true)

	for listen.run.Load() {
		select {
		case event, ok := <-listen.dispatchCh:
			if !ok {
				break
			}

			func(subscribes []Subscribe) {
				listen.subLock.RLock()
				defer listen.subLock.RUnlock()

				var ctx = context.Background()
				for _, sub := range subscribes {
					if sub.Matcher.Match(ctx, event.Key) && sub.Callback != nil {
						sub.Callback(event.Key, event.Payload)
					}
				}
			}(listen.subscribes)
		}
	}
	return nil
}

func (listen *Listener) Close() error {
	listen.run.Store(false)
	close(listen.dispatchCh)
	return nil
}

func (listen *Listener) Dispatch(key string, payload interface{}) {
	if !listen.run.Load() {
		return
	}
	listen.dispatchCh <- DispatchEvent{Key: key, Payload: payload}
}

func (listen *Listener) Watch(prefix string, fn SubscribeFunc) {
	listen.subLock.Lock()
	defer listen.subLock.Unlock()

	listen.subscribes = append(listen.subscribes, Subscribe{Matcher: &KeyMatch{Pattern: prefix}, Callback: fn})
}

func (listen *Listener) Unwatch(prefix string, fn SubscribeFunc) {

}

package edgekv

import "fmt"

type Message struct {
}

type MessageQueue interface {
	Publish(topic string, msg Message) error
	Subscribe(topic string, fn func(msg Message) error) error
}

type OpenQueueFunc func(args ...string) (MessageQueue, error)

var mqs = make(map[string]OpenQueueFunc)

func RegisterQueue(name string, opener OpenQueueFunc) {
	if _, ok := mqs[name]; ok {
		return
	}
	mqs[name] = opener
}

func OpenQueue(name string, args ...string) (MessageQueue, error) {
	if opener, ok := mqs[name]; !ok {
		return nil, fmt.Errorf("not found queue type %s", name)
	} else {
		return opener(args...)
	}
}

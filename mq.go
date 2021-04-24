package edgekv

import (
	"fmt"

	"github.com/r3labs/diff/v2"
)

var (
	PrefixTopic  = "edgekv"
	TopicPattern = PrefixTopic + "/{{ .EdgeID }}"
)

type Command string

const (
	CmdChangelog Command = "changelog"
)

type Message struct {
	From    string
	Type    Command
	Payload interface{}
}

type MessageChangelog struct {
	Key     string
	Changes diff.Changelog
}

type MessageQueue interface {
	Publish(topic string, msg Message) error
	Subscribe(topic string, fn func(msg Message) error) error
	Close() error
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

func (msg *Message) Build() bool {
	switch msg.Type {
	case CmdChangelog:
		msg.Payload = MessageChangelog{}
	default:
		return false
	}
	return true
}

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

// func (msg *Message) UnmarshalJSON(b []byte) error {
// 	var (
// 		head msgHead
// 		err  error
// 	)

// 	if err = json.Unmarshal(b, &head); err != nil {
// 		return err
// 	}

// 	msg.From = head.From
// 	msg.Type = head.Type

// 	switch head.Type {
// 	case CmdChangelog:
// 		var change MessageChangelog
// 		if err = utils.Unmarshal(head.Payload, &change); err != nil {
// 			return err
// 		}
// 		msg.Payload = change
// 	default:
// 		return errors.New("invalid message type")
// 	}

// 	return nil
// }

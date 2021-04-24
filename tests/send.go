package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hysios/edgekv"
	"github.com/r3labs/diff/v2"
)

type Message struct {
	From    string
	Type    string
	Payload interface{}
}

type msgHead struct {
	From    string
	Type    string
	Payload json.RawMessage
}

type MessageChangelog struct {
	Key     string
	Changes Changelog
}

type Changelog struct {
}

func main() {
	var (
		buf bytes.Buffer
		enc = gob.NewEncoder(&buf)
		key = "test.createdAt"
	)
	gob.Register(new(edgekv.MessageChangelog))

	msg := edgekv.Message{
		From: "ABE1234",
		Type: "changes",
		Payload: edgekv.MessageChangelog{
			Key:     key,
			Changes: diff.Changelog{},
		},
	}

	fmt.Println(enc.Encode(msg))
	ioutil.WriteFile("out.bin", buf.Bytes(), os.ModePerm)
}

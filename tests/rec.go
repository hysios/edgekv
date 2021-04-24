package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/hysios/edgekv"
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
		buf *bytes.Buffer
		out edgekv.Message
	)
	// gob.Register(new(MessageChangelog))
	b, _ := ioutil.ReadFile("out.bin")
	buf = bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	fmt.Println(dec.Decode(&out))
	fmt.Printf("out %#v", out)
}

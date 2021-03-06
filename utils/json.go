package utils

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"time"

	"github.com/hysios/edgekv"
)

func Stringify(val interface{}) string {
	b, _ := json.Marshal(val)
	return string(b)
}

func JSON(raw string) interface{} {
	var val = new(interface{})
	json.Unmarshal([]byte(raw), val)
	return *val
}

func Marshal(val Any) ([]byte, error) {
	var (
		buf bytes.Buffer
		enc = gob.NewEncoder(&buf)
		err error
	)
	if err = enc.Encode(val); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type Any = interface{}

func Unmarshal(b []byte, val Any) error {
	var (
		buf = bytes.NewBuffer(b)
		dec = gob.NewDecoder(buf)
	)

	return dec.Decode(val)
}

func init() {
	gob.Register(time.Time{})
	gob.Register(map[string]interface{}{})
	gob.Register(new(time.Duration))
	gob.Register([]interface{}{})
	gob.Register(new(interface{}))
	gob.Register(new(edgekv.MessageChangelog))
	gob.Register(new(edgekv.Message))
	gob.Register(new(Any))
}

package edgeserve

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"strconv"
)

func readBody(rd io.ReadCloser) []byte {
	b, _ := ioutil.ReadAll(rd)
	return b
}

func unquote(s string) string {
	qs, _ := strconv.Unquote(s)
	return qs
}

func quote(s string) string {
	return strconv.Quote(s)
}

func atob(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(s)
	return b
}

func btoa(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

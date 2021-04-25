package edgekv

type WatchEvent struct {
	Key  string
	From EdgeID
	Old  interface{}
	Val  interface{}
	Done func(bool)
}

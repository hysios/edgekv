package utils

import "encoding/json"

func Stringify(val interface{}) string {
	b, _ := json.Marshal(val)
	return string(b)
}

func JSON(raw string) interface{} {
	var val = new(interface{})
	json.Unmarshal([]byte(raw), val)
	return *val
}

package edgekv

import (
	"strings"

	"github.com/hysios/utils"
)

func SplitKey(key string) (prefix, field string) {
	ss := strings.SplitN(key, ".", 2)
	if len(ss) < 2 {
		return ss[0], ""
	}

	return ss[0], ss[1]
}

func SplitLastKey(key string) (prefix, field string) {
	keys := utils.SplitLast(key, '.', 2)
	if len(keys) < 2 {
		return keys[0], ""
	}
	return keys[0], keys[1]
}

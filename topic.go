package edgekv

import "fmt"

func Edgekey(edgeID EdgeID, key string) string {
	return fmt.Sprintf("%s:%s", edgeID, key)
}

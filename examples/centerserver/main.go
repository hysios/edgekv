package main

import (
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/center"
	"github.com/hysios/log"

	_ "github.com/hysios/edgekv/mq/mqtt"
	_ "github.com/hysios/edgekv/store/redis"
)

func main() {

	store, err := center.OpenCenterStore("redis")
	LogFatalf(err)
	mq, err := edgekv.OpenQueue("mqtt", "mqtt://127.0.0.1:1883/edgekv")
	LogFatalf(err)
	defer mq.Close()

	center.SetStore(store)
	center.SetMessageQueue(mq)
	edge, err := center.OpenEdge("OnlyTest")
	LogFatalf(err)

	edge.Watch("test.*", func(key string, old, new interface{}) error {
		log.Infof("watch %v", key)
		return nil
	})

	center.WatchEdges("test.*", func(key string, edgeID edgekv.EdgeID, old, new interface{}) error {
		log.Infof("watch from %s => %v", edgeID, key)
		return nil
	})

	log.Infof("start Edgekv Center server ")
	LogFatalf(center.StartServer())
}

func LogFatalf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

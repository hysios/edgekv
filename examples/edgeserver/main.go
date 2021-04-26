package main

import (
	"time"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/edge"
	"github.com/hysios/edgekv/edge/edgeserve"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"

	_ "github.com/hysios/edgekv/mq/mqtt"
	_ "github.com/hysios/edgekv/store/buntdb"
)

const ClientID = "OnlyTest"

func main() {
	store, _ := edgekv.OpenStore("buntdb", "edgekv.db")
	mq, _ := edgekv.OpenQueue("mqtt", "mqtt://127.0.0.1:1883/edgekv?client_id="+ClientID)

	edgeserve.SetStore(store)
	edgeserve.SetMessageQueue(mq)
	edgeserve.SetEdgeID(ClientID)

	go func() {
		log.Fatal(edgeserve.StartUnix("/tmp/edgekv.sock"))
	}()

	time.Sleep(500 * time.Millisecond)
	edgestore, err := edge.Open()
	utils.LogFatalf(err)
	edgestore.Watch("test.*", func(key string, old, new interface{}) error {
		log.Infof("watch key '%s' chagne => '%v'", key, new)
		return nil
	})

	ch := make(chan bool)
	<-ch
}

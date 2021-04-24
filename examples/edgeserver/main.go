package main

import (
	"log"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/edge/edgeserve"
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

	log.Fatalln(edgeserve.StartUnix("/tmp/edgekv.sock"))
}

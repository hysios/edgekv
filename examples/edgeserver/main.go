package main

import (
	"log"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/edge/edgeserve"
	_ "github.com/hysios/edgekv/store/memory"
)

func main() {
	store, _ := edgekv.OpenStore("memory")
	edgeserve.SetStore(store)

	log.Fatalln(edgeserve.StartUnix("/tmp/edgekv.sock"))
}

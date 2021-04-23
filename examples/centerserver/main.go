package main

import (
	"log"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/center"
	_ "github.com/hysios/edgekv/store/redis"
)

func main() {
	store, _ := center.OpenCenterStore("redis")
	mq, _ := edgekv.OpenQueue("mqtt", "mqtt://127.0.0.1:1883/edgekv")
	center.SetStore(store)
	center.SetMessageQueue(mq)

	log.Fatalln(center.StartServer())
}

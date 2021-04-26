package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/center"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
	. "github.com/hysios/utils/response"

	_ "github.com/hysios/edgekv/mq/mqtt"
	_ "github.com/hysios/edgekv/store/redis"
)

func main() {

	store, err := center.OpenCenterStore("redis")
	utils.LogFatalf(err)
	mq, err := edgekv.OpenQueue("mqtt", "mqtt://127.0.0.1:1883/edgekv")
	utils.LogFatalf(err)

	defer func() {
		mq.Close()
		center.Stop()
	}()

	center.SetStore(store)
	center.SetMessageQueue(mq)
	edge, err := center.OpenEdge("OnlyTest")
	utils.LogFatalf(err)

	edge.Watch("test.*", func(key string, old, new interface{}) error {
		log.Infof("watch %v", key)
		return nil
	})

	center.WatchEdges("test.*", func(key string, edgeID edgekv.EdgeID, old, new interface{}) error {
		log.Infof("watch from %s => %v", edgeID, key)
		return nil
	})

	r := mux.NewRouter()
	r.Path("/{edgeID}/{key}").
		Methods("GET").
		HandlerFunc(GetKey)

	r.Path("/{edgeID}/{key}").
		Methods("POST").
		HandlerFunc(SetKey)

	log.Infof("start Edgekv Center server ")
	go func() {
		utils.LogFatalf(center.StartServer())
	}()

	log.Fatal(http.ListenAndServe(":9097", r))
}

func GetKey(w http.ResponseWriter, r *http.Request) {
	var (
		params = mux.Vars(r)
		key    = params["key"]
		edgeID = params["edgeID"]
	)
	edge, err := center.OpenEdge(edgekv.EdgeID(edgeID))
	if err != nil {
		AbortErr(w, http.StatusNotFound, err)
		return
	}

	if val, ok := edge.Get(key); ok {
		log.Infof("get '%s' value => %v", key, val)
		Jsonify(w, &map[string]interface{}{"data": val})
	} else {
		AbortErr(w, http.StatusNotFound, fmt.Errorf("not found key '%s'", key))
		return
	}
}

func SetKey(w http.ResponseWriter, r *http.Request) {
	var (
		params = mux.Vars(r)
		key    = params["key"]
		edgeID = params["edgeID"]
		val    = new(interface{})
	)
	edge, err := center.OpenEdge(edgekv.EdgeID(edgeID))
	if err != nil {
		AbortErr(w, http.StatusNotFound, err)
		return
	}

	var dec = json.NewDecoder(r.Body)
	if err = dec.Decode(val); err != nil {
		AbortErr(w, http.StatusBadRequest, err)
		return
	}

	edge.Set(key, *val)
	log.Infof("set '%s' value => %v", key, *val)

	Jsonify(w, nil)
}

package edge

import (
	"net/http"

	"github.com/hysios/edgekv"
	"github.com/hysios/edgekv/stream"
	"github.com/hysios/edgekv/utils"
	"github.com/hysios/log"
)

type MsgFunc func(msg edgekv.Message)

type MsgStream struct {
	Streamer
}

type Streamer interface {
	Message(fn stream.MessageFunc)
	Send([]byte) (int, error)
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*MsgStream, error) {
	stream, err := stream.Upgrade(w, r)
	if err != nil {
		return nil, err
	}

	return &MsgStream{Streamer: stream}, nil
}

func Connect(uri string) (*MsgStream, error) {
	client, err := stream.NewClient(uri)
	if err != nil {
		return nil, err
	}
	return &MsgStream{Streamer: client}, nil
}

func (rece *MsgStream) MessageMsg(fn MsgFunc) {
	rece.Message(func(mt stream.MessageType, msg []byte) {
		var data EdgeData
		if mt == stream.StBinary {
			if err := utils.Unmarshal(msg, &data); err != nil {
				log.Errorf("msgstream: unmarshal gob error %s", err)
			}

			if msg, ok := data.Data.(edgekv.Message); ok {
				fn(msg)
			}
		}
	})
}

func (rece *MsgStream) SendMsg(msg edgekv.Message) error {
	var data = EdgeData{
		Data:   msg,
		Status: "success",
	}
	if b, err := utils.Marshal(data); err != nil {
		return err
	} else {
		rece.Send(b)
	}

	return nil
}

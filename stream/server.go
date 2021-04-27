package stream

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Server struct {
	*Client
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*Server, error) {
	var serve = &Server{}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	serve.Client = &Client{conn: conn, send: make(chan []byte, 256)}
	go serve.readPump()
	go serve.writePump()
	return serve, nil
}

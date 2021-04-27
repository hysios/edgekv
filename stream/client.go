package stream

import (
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hysios/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096 * 10
)

type MessageType int

const (
	StBinary MessageType = iota
	StText
)

type MessageFunc func(typ MessageType, msg []byte)

type Client struct {
	URL url.URL

	conn    *websocket.Conn
	send    chan []byte
	handler MessageFunc
}

func NewClient(uri string) (*Client, error) {
	var client = &Client{}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	client.URL = *u
	client.send = make(chan []byte, 256)
	if err = client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (client *Client) Connect() error {
	log.Infof("connecting to %s", client.URL.String())

	c, _, err := websocket.DefaultDialer.Dial(client.URL.String(), nil)
	if err != nil {
		return fmt.Errorf("stream_client: dial error: %w", err)
	}

	client.conn = c
	// client.send = make(chan []byte, 256)
	go client.readPump()
	go client.writePump()
	return nil
}

func (client *Client) Close() error {
	return client.conn.Close()
}

var mtConverts = map[int]MessageType{
	1: StText,
	2: StBinary,
}

func (client *Client) readPump() {
	defer func() {
		// client.hub.unregister <- c
		client.conn.Close()
	}()
	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error { client.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		mt, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		if client.handler != nil {
			client.handler(mtConverts[mt], message)
		}
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()
	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(client.send)
			for i := 0; i < n; i++ {
				// w.Write(newline)
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (client *Client) Send(b []byte) (int, error) {
	client.send <- b
	return len(b), nil
}

func (client *Client) Message(fn MessageFunc) {
	client.handler = fn
}

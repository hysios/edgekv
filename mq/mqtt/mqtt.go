package mqtt

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hysios/edgekv"
	"github.com/hysios/log"
	"github.com/kr/pretty"
)

type mqttMQ struct {
	Prefix string

	Q        byte
	mqClient mqtt.Client
}

var (
	ClientID = "TEST"
	QosMode  = 2
)

func SetClientID(clientID string) {
	ClientID = clientID
}

// OpenMqttMQ 打开消息队列
func OpenMqttMQ(uri string) (*mqttMQ, error) {
	var (
		opts *mqtt.ClientOptions
		mq   = &mqttMQ{
			Q: byte(QosMode),
		}
		err error
	)

	if opts, err = mq.ParseURI(uri); err != nil {
		return nil, err
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, err
	}
	mq.mqClient = client

	return mq, nil
}

func (mq *mqttMQ) ParseURI(uri string) (*mqtt.ClientOptions, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("mqtt_mq: parse uri error: %w", err)
	}

	log.Infof("mqtt: open mqtt at %s", uri)
	var opts = mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", u.Host))
	opts.SetUsername(u.User.Username())
	if pass, ok := u.User.Password(); ok {
		opts.SetPassword(pass)
	}
	if len(u.Path) > 1 {
		mq.Prefix = strings.TrimPrefix(u.Path, "/")
	}
	opts.SetDefaultPublishHandler(mq.messagePubHandler)
	opts.OnConnect = mq.connectHandler
	opts.OnConnectionLost = mq.connectLostHandler
	mq.parseQuery(opts, u.Query())

	if len(opts.ClientID) == 0 {
		opts.SetClientID(ClientID)
	}
	return opts, nil
}

func (mq *mqttMQ) parseQuery(opts *mqtt.ClientOptions, q url.Values) {

	for key := range q {
		switch key {
		case "auto_reconnect":
			v, _ := strconv.ParseBool(q.Get(key))
			opts.AutoReconnect = v
		case "timeout":
			dt, _ := time.ParseDuration(q.Get(key))
			opts.SetConnectTimeout(dt)
			opts.SetPingTimeout(dt)
			opts.SetWriteTimeout(dt)
		case "client_id":
			opts.SetClientID(q.Get(key))
		case "clean_session":
			v, _ := strconv.ParseBool(q.Get(key))
			opts.SetCleanSession(v)
		}
	}
}

func (mq *mqttMQ) Publish(topic string, msg edgekv.Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Debugf("mqtt: publish to topic %s with qos mode %d", mq.FullTopic(topic), mq.Q)
	tok := mq.mqClient.Publish(mq.FullTopic(topic), mq.Q, false, b)

	return mq.Wait(tok)
}

func (mq *mqttMQ) Wait(tok mqtt.Token) error {
	if tok.Wait() && tok.Error() != nil {
		return tok.Error()
	}

	return nil
}

func (mq *mqttMQ) FullTopic(_topic string) string {
	return path.Join(mq.Prefix, _topic)
}

func (mq *mqttMQ) Subscribe(topic string, fn func(msg edgekv.Message) error) error {
	log.Infof("subscribe topic %s with qos mode %d", mq.FullTopic(topic), mq.Q)
	tok := mq.mqClient.Subscribe(mq.FullTopic(topic), mq.Q, func(_ mqtt.Client, rawmsg mqtt.Message) {
		var (
			msg     edgekv.Message
			payload = rawmsg.Payload()
			err     error
		)

		if err = json.Unmarshal(payload, &msg); err != nil {
			log.Errorf("mqtt: unmarshal message error %s", err)
		}

		log.Debugf("msg % #v", pretty.Formatter(msg))
		if err = fn(msg); err == nil {
			rawmsg.Ack()
		}
	})
	return mq.Wait(tok)
}

func (mq *mqttMQ) connectHandler(client mqtt.Client) {
	log.Debug("Connected")
}

func (mq *mqttMQ) messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	log.Debugf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func (mq *mqttMQ) connectLostHandler(client mqtt.Client, err error) {
	log.Debugf("Connect lost: %v", err)
}

func init() {
	edgekv.RegisterQueue("mqtt", func(args ...string) (edgekv.MessageQueue, error) {
		return OpenMqttMQ(args[0])
	})
}

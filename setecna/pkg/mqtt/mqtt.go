// Package mqtt wraps the paho MQTT client with sane defaults for the
// Setecna add-on: automatic reconnection, Last Will & Testament for
// availability tracking and retained publishing helpers.
package mqtt

import (
	"fmt"
	"log/slog"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Message represents a single MQTT message to publish.
type Message struct {
	Topic   string
	Payload string
	Qos     byte
	Retain  bool
}

// Messages is a batch of MQTT messages.
type Messages []Message

// Client wraps the underlying paho client.
type Client struct {
	client            paho.Client
	availabilityTopic string
	onConnect         func(c *Client)
}

// Options holds the connection parameters for the broker.
type Options struct {
	Host              string
	Port              string
	Username          string
	Password          string
	ClientID          string
	AvailabilityTopic string
	// OnConnect is invoked on every (re)connection, after the
	// availability "online" message has been published. Use it to
	// (re)subscribe and republish discovery/state.
	OnConnect func(c *Client)
}

// Connect establishes the connection to the broker and blocks until the
// first connection succeeds or the timeout expires.
func Connect(o Options) (*Client, error) {
	c := &Client{availabilityTopic: o.AvailabilityTopic, onConnect: o.OnConnect}

	opts := paho.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%s", o.Host, o.Port)).
		SetUsername(o.Username).
		SetPassword(o.Password).
		SetClientID(o.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetMaxReconnectInterval(2 * time.Minute).
		SetKeepAlive(30 * time.Second).
		SetOrderMatters(false)

	// Last Will: mark the device unavailable if the add-on dies.
	if o.AvailabilityTopic != "" {
		opts.SetWill(o.AvailabilityTopic, "offline", 1, true)
	}

	opts.SetOnConnectHandler(func(pc paho.Client) {
		slog.Info("connected to MQTT broker", "host", o.Host, "port", o.Port)
		if c.availabilityTopic != "" {
			pc.Publish(c.availabilityTopic, 1, true, "online")
		}
		if c.onConnect != nil {
			// Run in a goroutine: paho forbids blocking the handler.
			go c.onConnect(c)
		}
	})
	opts.SetConnectionLostHandler(func(_ paho.Client, err error) {
		slog.Warn("MQTT connection lost, reconnecting", "error", err)
	})

	c.client = paho.NewClient(opts)
	token := c.client.Connect()
	if !token.WaitTimeout(60 * time.Second) {
		return nil, fmt.Errorf("timeout connecting to MQTT broker %s:%s", o.Host, o.Port)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("connecting to MQTT broker: %w", err)
	}
	return c, nil
}

// Disconnect gracefully closes the connection, publishing the offline
// availability message first.
func (c *Client) Disconnect() {
	if c.availabilityTopic != "" {
		t := c.client.Publish(c.availabilityTopic, 1, true, "offline")
		t.WaitTimeout(2 * time.Second)
	}
	c.client.Disconnect(500)
}

// Publish sends a single message and waits for completion.
func (c *Client) Publish(m Message) error {
	token := c.client.Publish(m.Topic, m.Qos, m.Retain, m.Payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("timeout publishing to %s", m.Topic)
	}
	return token.Error()
}

// BatchPublish sends a batch of messages sequentially.
func (c *Client) BatchPublish(msgs Messages) {
	for _, m := range msgs {
		if err := c.Publish(m); err != nil {
			slog.Error("publish failed", "topic", m.Topic, "error", err)
		}
	}
}

// Subscribe registers a handler for a topic filter.
func (c *Client) Subscribe(filter string, qos byte, handler paho.MessageHandler) error {
	token := c.client.Subscribe(filter, qos, handler)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("timeout subscribing to %s", filter)
	}
	return token.Error()
}

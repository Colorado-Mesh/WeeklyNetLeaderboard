package mqtt

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Message struct {
	Topic      string
	PayloadHex string
	ObservedAt time.Time
}

type Consumer struct {
	client          paho.Client
	logger          *slog.Logger
	maxPayloadBytes int
	mu              sync.Mutex
	subs            []subscription
}

type subscription struct {
	topic   string
	handler func(context.Context, Message)
}

func NewConsumer(logger *slog.Logger, brokerURL, clientID, username, password string, maxPayloadBytes int) *Consumer {
	c := &Consumer{
		logger:          logger,
		maxPayloadBytes: maxPayloadBytes,
	}
	opts := paho.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(3 * time.Second)
	opts.OnConnect = func(client paho.Client) {
		logger.Info("mqtt connected")
		c.resubscribeAll(client)
	}
	opts.OnConnectionLost = func(_ paho.Client, err error) {
		if err == nil {
			logger.Warn("mqtt connection lost")
			return
		}
		logger.Warn("mqtt connection lost", "error", err.Error())
	}

	c.client = paho.NewClient(opts)
	return c
}

func (c *Consumer) Connect(ctx context.Context) error {
	if c.client.IsConnectionOpen() {
		return nil
	}
	token := c.client.Connect()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		if err := token.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Consumer) Close() {
	if c.client.IsConnectionOpen() {
		c.client.Disconnect(250)
	}
}

func (c *Consumer) Subscribe(topic string, handler func(context.Context, Message)) error {
	sub := subscription{topic: topic, handler: handler}
	c.mu.Lock()
	c.subs = append(c.subs, sub)
	c.mu.Unlock()

	if c.client.IsConnectionOpen() {
		if err := c.subscribeWithClient(c.client, sub); err != nil {
			return err
		}
	}
	return nil
}

func (c *Consumer) subscribeWithClient(client paho.Client, sub subscription) error {
	token := client.Subscribe(sub.topic, 1, func(_ paho.Client, msg paho.Message) {
		message, ok := c.buildMessage(msg.Topic(), msg.Payload())
		if !ok {
			return
		}
		sub.handler(context.Background(), message)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe %s: %w", sub.topic, err)
	}
	c.logger.Info("mqtt subscribed", "topic", sub.topic)
	return nil
}

func (c *Consumer) buildMessage(topic string, payload []byte) (Message, bool) {
	if len(payload) > c.maxPayloadBytes {
		c.logger.Warn(
			"mqtt payload dropped: too large",
			"topic",
			topic,
			"payload_bytes",
			len(payload),
			"max_payload_bytes",
			c.maxPayloadBytes,
		)
		return Message{}, false
	}
	return Message{
		Topic:      topic,
		PayloadHex: strings.TrimSpace(string(payload)),
		ObservedAt: time.Now().UTC(),
	}, true
}

func (c *Consumer) resubscribeAll(client paho.Client) {
	c.mu.Lock()
	subs := append([]subscription(nil), c.subs...)
	c.mu.Unlock()
	for _, sub := range subs {
		if err := c.subscribeWithClient(client, sub); err != nil {
			c.logger.Warn("mqtt resubscribe failed", "topic", sub.topic, "error", err.Error())
		}
	}
}

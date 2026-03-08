// Package mqtt provides an MQTT client wrapper with publisher and ACK handling.
package mqtt

import (
	"encoding/json"
	"fmt"
	"time"

	"smartbed/internal/config"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

// Client wraps the Paho MQTT client.
type Client struct {
	inner paho.Client
	cfg   *config.Config
	ackCh chan ACKPayload
}

// ACKPayload represents a device acknowledgement message.
type ACKPayload struct {
	CommandID string    `json:"command_id"`
	DeviceID  string    `json:"device_id"`
	Status    string    `json:"status"` // executed | failed
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates and connects an MQTT client.
func New(cfg *config.Config) (*Client, error) {
	ackCh := make(chan ACKPayload, 256)

	opts := paho.NewClientOptions().
		AddBroker(cfg.MQTT.Broker).
		SetClientID(cfg.MQTT.ClientID).
		SetUsername(cfg.MQTT.Username).
		SetPassword(cfg.MQTT.Password).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetOnConnectHandler(func(c paho.Client) {
			log.Info().Str("broker", cfg.MQTT.Broker).Msg("MQTT connected")
			// Subscribe to all device ACK topics on reconnect.
			token := c.Subscribe("devices/+/ack", cfg.MQTT.QOS, func(_ paho.Client, msg paho.Message) {
				var ack ACKPayload
				if err := json.Unmarshal(msg.Payload(), &ack); err != nil {
					log.Warn().Err(err).Msg("MQTT: failed to parse ACK payload")
					return
				}
				select {
				case ackCh <- ack:
				default:
					log.Warn().Msg("MQTT: ACK channel full, dropping message")
				}
			})
			token.Wait()
			if err := token.Error(); err != nil {
				log.Error().Err(err).Msg("MQTT: subscribe to ACK topic failed")
			}
		}).
		SetConnectionLostHandler(func(_ paho.Client, err error) {
			log.Warn().Err(err).Msg("MQTT connection lost")
		})

	c := paho.NewClient(opts)
	token := c.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &Client{inner: c, cfg: cfg, ackCh: ackCh}, nil
}

// Publish sends a message to the given MQTT topic.
func (c *Client) Publish(topic string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mqtt marshal: %w", err)
	}
	token := c.inner.Publish(topic, c.cfg.MQTT.QOS, false, data)
	token.Wait()
	return token.Error()
}

// CommandTopic returns the standard command topic for a device.
func CommandTopic(deviceID string) string {
	return fmt.Sprintf("devices/%s/commands", deviceID)
}

// ACKChannel returns the channel on which incoming ACKs are received.
func (c *Client) ACKChannel() <-chan ACKPayload {
	return c.ackCh
}

// Disconnect gracefully disconnects the client.
func (c *Client) Disconnect() {
	c.inner.Disconnect(500)
}

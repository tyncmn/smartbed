// Package worker – MQTT ACK worker.
// Listens for device ACK messages and updates command lifecycle.
package worker

import (
	"context"

	mqttclient "smartbed/internal/mqtt"
	"smartbed/internal/service"

	"github.com/rs/zerolog/log"
)

// MQTTACKWorker listens on the MQTT ACK channel and processes ACKs.
type MQTTACKWorker struct {
	mqtt      *mqttclient.Client
	deviceSvc *service.DeviceCommandService
}

// NewMQTTACKWorker creates a new MQTTACKWorker.
func NewMQTTACKWorker(mqtt *mqttclient.Client, deviceSvc *service.DeviceCommandService) *MQTTACKWorker {
	return &MQTTACKWorker{mqtt: mqtt, deviceSvc: deviceSvc}
}

// Run starts consuming ACK messages until ctx is cancelled.
func (w *MQTTACKWorker) Run(ctx context.Context) {
	log.Info().Msg("MQTT ACK worker started")
	ackCh := w.mqtt.ACKChannel()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("MQTT ACK worker stopped")
			return
		case ack, ok := <-ackCh:
			if !ok {
				return
			}
			w.deviceSvc.HandleACK(ctx, ack)
		}
	}
}

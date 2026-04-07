package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/naviNBRuas/APA/pkg/networking"
)

// controllerTransport reuses the controller communication topic for control-plane gossip.
type controllerTransport struct {
	logger     *slog.Logger
	p2p        *networking.P2P
	idProvider func() string
}

// NewControllerTransport wraps a P2P instance for control-plane messaging.
// idProvider may be nil; when nil, the P2P host ID will be used.
func NewControllerTransport(logger *slog.Logger, p2p *networking.P2P, idProvider func() string) Transport {
	return &controllerTransport{logger: logger, p2p: p2p, idProvider: idProvider}
}

func (t *controllerTransport) Publish(ctx context.Context, topic string, payload []byte) error {
	msg := networking.ControllerMessage{Type: topic, Data: payload}
	if err := t.p2p.PublishControllerMessage(ctx, mustJSON(msg)); err != nil {
		return fmt.Errorf("publish controller message: %w", err)
	}
	return nil
}

func (t *controllerTransport) Subscribe(ctx context.Context, topic string) (<-chan []byte, error) {
	ch, err := t.p2p.SubscribeControllerMessages(ctx)
	if err != nil {
		return nil, err
	}
	out := make(chan []byte, 16)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if msg.Type != topic {
					continue
				}
				out <- msg.Data
			}
		}
	}()
	return out, nil
}

func (t *controllerTransport) LocalID() string {
	if t.idProvider != nil {
		if id := t.idProvider(); id != "" {
			return id
		}
	}
	return t.p2p.HostID()
}

func mustJSON(msg networking.ControllerMessage) []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

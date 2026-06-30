package networking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/naviNBRuas/APA/pkg/module"
)

// JoinHeartbeatTopic joins the heartbeat topic.
func (p *P2P) JoinHeartbeatTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(HeartbeatTopic)
	if err != nil {
		return fmt.Errorf("failed to join heartbeat topic: %w", err)
	}

	p.heartbeatTopic = topic
	return nil
}

// IsHeartbeatJoined reports whether the heartbeat topic is active.
func (p *P2P) IsHeartbeatJoined() bool {
	return p != nil && p.heartbeatTopic != nil
}

// StartHeartbeat starts broadcasting heartbeats.
func (p *P2P) StartHeartbeat(ctx context.Context, interval time.Duration) {
	if p.heartbeatTopic == nil {
		p.logger.Error("Heartbeat topic not joined")
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Stopping heartbeat")
			return
		case <-ticker.C:
			msg := map[string]interface{}{
				"peer_id": p.host.ID().String(),
				"time":    time.Now().Unix(),
			}

			msgBytes, err := json.Marshal(msg)
			if err != nil {
				p.logger.Error("Failed to marshal heartbeat message", "error", err)
				continue
			}

			if err := p.heartbeatTopic.Publish(ctx, msgBytes); err != nil {
				p.logger.Error("Failed to publish heartbeat", "error", err)
			}
		}
	}
}

// JoinModuleTopic joins the module announcement topic.
func (p *P2P) JoinModuleTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(ModuleTopic)
	if err != nil {
		return fmt.Errorf("failed to join module topic: %w", err)
	}

	p.moduleTopic = topic
	return nil
}

// AnnounceModule announces a module to the network.
func (p *P2P) AnnounceModule(ctx context.Context, manifest module.Manifest) error {
	if p.moduleTopic == nil {
		return fmt.Errorf("module topic not joined")
	}

	msg := ModuleAnnouncementMessage{
		Manifest:        manifest,
		AnnouncerPeerID: p.host.ID().String(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal module announcement: %w", err)
	}

	if err := publishWithRetry(ctx, p.logger, p.moduleTopic, msgBytes, "module announcement"); err != nil {
		return err
	}

	p.logger.Info("Announced module", "name", manifest.Name, "version", manifest.Version)
	return nil
}

// JoinControllerCommTopic joins the controller communication topic.
func (p *P2P) JoinControllerCommTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(ControllerCommTopic)
	if err != nil {
		return fmt.Errorf("failed to join controller communication topic: %w", err)
	}

	p.controllerCommTopic = topic
	return nil
}

// IsControllerJoined reports whether the controller communication topic is active.
func (p *P2P) IsControllerJoined() bool {
	return p != nil && p.controllerCommTopic != nil
}

// PublishControllerMessage publishes a controller message to the network.
func (p *P2P) PublishControllerMessage(ctx context.Context, msgBytes []byte) error {
	if p.controllerCommTopic == nil {
		return fmt.Errorf("controller communication topic not joined")
	}

	return publishWithRetry(ctx, p.logger, p.controllerCommTopic, msgBytes, "controller message")
}

// SubscribeControllerMessages subscribes to controller messages.
func (p *P2P) SubscribeControllerMessages(ctx context.Context) (<-chan *ControllerMessage, error) {
	if p.controllerCommTopic == nil {
		return nil, fmt.Errorf("controller communication topic not joined")
	}

	sub, err := p.controllerCommTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to controller messages: %w", err)
	}

	msgCh := make(chan *ControllerMessage, 10)

	go func() {
		defer close(msgCh)
		defer sub.Cancel()

		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, pubsub.ErrSubscriptionCancelled) {
					return
				}
				p.logger.Error("Failed to read controller message", "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
				}
				continue
			}

			if msg == nil {
				continue
			}

			var ctrlMsg ControllerMessage
			if err := json.Unmarshal(msg.Data, &ctrlMsg); err != nil {
				p.logger.Error("Failed to unmarshal controller message", "error", err)
				continue
			}

			peerID, err := peer.IDFromBytes(msg.From)
			if err != nil {
				p.logger.Error("Failed to decode peer ID from message", "error", err)
				continue
			}
			ctrlMsg.SenderPeerID = peerID.String()

			select {
			case msgCh <- &ctrlMsg:
			default:
				p.logger.Warn("Controller message channel full, dropping message")
			}
		}
	}()

	return msgCh, nil
}

// JoinLeaderElectionTopic joins the leader election topic.
func (p *P2P) JoinLeaderElectionTopic(ctx context.Context) error {
	topic, err := p.pubsub.Join(LeaderElectionTopic)
	if err != nil {
		return fmt.Errorf("failed to join leader election topic: %w", err)
	}

	p.leaderElectionTopic = topic
	return nil
}

// IsLeaderElectionJoined reports whether the leader election topic is active.
func (p *P2P) IsLeaderElectionJoined() bool {
	return p != nil && p.leaderElectionTopic != nil
}

// SubscribeLeaderElectionMessages subscribes to leader election messages.
func (p *P2P) SubscribeLeaderElectionMessages(ctx context.Context) (<-chan *LeaderElectionMessage, error) {
	if p.leaderElectionTopic == nil {
		return nil, fmt.Errorf("leader election topic not joined")
	}

	sub, err := p.leaderElectionTopic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to leader election messages: %w", err)
	}

	msgCh := make(chan *LeaderElectionMessage, 10)

	go func() {
		defer close(msgCh)
		defer sub.Cancel()

		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, pubsub.ErrSubscriptionCancelled) {
					return
				}
				p.logger.Error("Failed to read leader election message", "error", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
				}
				continue
			}

			if msg == nil {
				continue
			}

			var leMsg LeaderElectionMessage
			if err := json.Unmarshal(msg.Data, &leMsg); err != nil {
				p.logger.Error("Failed to unmarshal leader election message", "error", err)
				continue
			}

			peerID, err := peer.IDFromBytes(msg.From)
			if err != nil {
				p.logger.Error("Failed to decode peer ID from message", "error", err)
				continue
			}
			leMsg.CandidateID = peerID.String()
			leMsg.Timestamp = time.Now()

			select {
			case msgCh <- &leMsg:
			default:
				p.logger.Warn("Leader election message channel full, dropping message")
			}
		}
	}()

	return msgCh, nil
}

// PublishLeaderElectionMessage publishes a leader election message.
func (p *P2P) PublishLeaderElectionMessage(ctx context.Context, msg LeaderElectionMessage) error {
	if p.leaderElectionTopic == nil {
		return fmt.Errorf("leader election topic not joined")
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal leader election message: %w", err)
	}

	return publishWithRetry(ctx, p.logger, p.leaderElectionTopic, msgBytes, "leader election message")
}

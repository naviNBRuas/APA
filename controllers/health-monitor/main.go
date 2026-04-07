package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// HealthStatus represents a lightweight health snapshot.
type HealthStatus struct {
	Controller string    `json:"controller"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
	Details    string    `json:"details"`
}

// ControllerMessage mirrors the APA controller message envelope.
type ControllerMessage struct {
	Type         string          `json:"type"`
	SenderPeerID string          `json:"sender_peer_id"`
	Payload      json.RawMessage `json:"payload"`
	Timestamp    time.Time       `json:"timestamp"`
}

func main() {
	configFile := flag.String("config", "", "Path to configuration file")
	messageFile := flag.String("message-file", "", "Path to message file")
	flag.Parse()

	log.Printf("Health-monitor controller starting (config=%s, message=%s)", *configFile, *messageFile)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for lifecycle and config reload.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)

	go runHeartbeat(ctx)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGHUP:
				log.Printf("Health-monitor: reload requested (config=%s)", *configFile)
			case syscall.SIGUSR1:
				if err := processMessage(*messageFile); err != nil {
					log.Printf("Health-monitor: message processing error: %v", err)
				}
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("Health-monitor: shutting down")
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func runHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			status := HealthStatus{
				Controller: "health-monitor",
				Timestamp:  t,
				Status:     "ok",
				Details:    "controller alive",
			}
			b, _ := json.Marshal(status)
			log.Printf("Health-monitor heartbeat: %s", string(b))
		}
	}
}

func processMessage(messageFile string) error {
	if messageFile == "" {
		return nil
	}

	data, err := os.ReadFile(messageFile)
	if err != nil {
		return err
	}

	var msg ControllerMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	log.Printf("Health-monitor received message type=%s from=%s", msg.Type, msg.SenderPeerID)
	return nil
}

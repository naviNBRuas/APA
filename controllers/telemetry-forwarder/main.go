package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// TelemetryEnvelope represents telemetry forwarded to an external sink.
type TelemetryEnvelope struct {
	Timestamp    time.Time       `json:"timestamp"`
	Controller   string          `json:"controller"`
	SenderPeerID string          `json:"sender_peer_id"`
	Type         string          `json:"type"`
	Payload      json.RawMessage `json:"payload"`
}

// ControllerMessage matches APA controller envelope.
type ControllerMessage struct {
	Type         string          `json:"type"`
	SenderPeerID string          `json:"sender_peer_id"`
	Payload      json.RawMessage `json:"payload"`
	Timestamp    time.Time       `json:"timestamp"`
}

func main() {
	configFile := flag.String("config", "", "Path to configuration file")
	messageFile := flag.String("message-file", "", "Path to message file")
	sinkURL := flag.String("sink", "http://localhost:8080/telemetry", "HTTP sink for telemetry")
	flag.Parse()

	log.Printf("Telemetry-forwarder starting (config=%s, message=%s, sink=%s)", *configFile, *messageFile, *sinkURL)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigs {
		switch sig {
		case syscall.SIGHUP:
			log.Printf("Telemetry-forwarder: reload requested (config=%s)", *configFile)
		case syscall.SIGUSR1:
			if err := processAndForward(*messageFile, *sinkURL); err != nil {
				log.Printf("Telemetry-forwarder: process error: %v", err)
			}
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("Telemetry-forwarder: shutting down")
			return
		}
	}
}

func processAndForward(messageFile, sink string) error {
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

	env := TelemetryEnvelope{
		Timestamp:    time.Now().UTC(),
		Controller:   "telemetry-forwarder",
		SenderPeerID: msg.SenderPeerID,
		Type:         msg.Type,
		Payload:      msg.Payload,
	}

	body, err := json.Marshal(env)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, sink, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("Telemetry-forwarder posted telemetry: status=%d bytes=%d", resp.StatusCode, len(body))
	return nil
}

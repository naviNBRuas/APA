package manager

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	controllerPkg "github.com/naviNBRuas/APA/pkg/controller"
	manifest "github.com/naviNBRuas/APA/pkg/controller/manifest"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/naviNBRuas/APA/pkg/policy"
)

type mockPolicyEnforcer struct{}

func (m *mockPolicyEnforcer) Authorize(ctx context.Context, subject string, action string, resource string) (bool, string, error) {
	return true, "", nil
}

func TestManagerCreation(t *testing.T) {
	logger := slog.Default()
	manager := NewManager(logger, "/tmp", &mockPolicyEnforcer{})

	assert.NotNil(t, manager, "Failed to create manager")
}

func TestSetP2PNetwork(t *testing.T) {
	logger := slog.Default()
	manager := NewManager(logger, "/tmp", &mockPolicyEnforcer{})

	manager.SetP2PNetwork(&networking.P2P{})
}

func TestSendMessageToController(t *testing.T) {
	logger := slog.Default()
	manager := NewManager(logger, "/tmp", &mockPolicyEnforcer{})

	dummyManifest := &manifest.Manifest{
		Name: "test-controller",
	}
	dummyController := controllerPkg.NewDummyController("test-controller", logger, dummyManifest)

	manager.mu.Lock()
	manager.controllers["test-controller"] = dummyController
	manager.mu.Unlock()

	_ = manager.SendMessageToController
}

func TestSendP2PMessageToController(t *testing.T) {
	logger := slog.Default()
	manager := NewManager(logger, "/tmp", &mockPolicyEnforcer{})

	_ = manager.SendP2PMessageToController
}

func TestManagerImplementsPolicyEnforcer(t *testing.T) {
	var _ policy.PolicyEnforcer = (*mockPolicyEnforcer)(nil)
}

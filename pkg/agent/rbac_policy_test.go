package agent

import (
	"context"
	"testing"

	"github.com/naviNBRuas/APA/pkg/opa"
	"github.com/stretchr/testify/require"
)

func TestRBACPolicy_AllowsHealthAndStatus(t *testing.T) {
	engine := newTestPolicyEngine(t)
	input := map[string]interface{}{
		"path":   "/admin/health",
		"method": "GET",
	}
	allowed, err := engine.Authorize(context.Background(), input)
	require.NoError(t, err)
	require.True(t, allowed)

	input = map[string]interface{}{
		"path":   "/admin/status",
		"method": "GET",
	}
	allowed, err = engine.Authorize(context.Background(), input)
	require.NoError(t, err)
	require.True(t, allowed)
}

func newTestPolicyEngine(t *testing.T) *opa.OPAPolicyEngine {
	e := opa.NewOPAPolicyEngine()
	err := e.LoadPolicy(context.Background(), "../../configs/admin_policy.rego")
	require.NoError(t, err)
	return e
}

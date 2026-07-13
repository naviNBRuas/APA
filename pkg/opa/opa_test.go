package opa

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewOPAPolicyEngine(t *testing.T) {
	engine := NewOPAPolicyEngine()
	require.NotNil(t, engine, "expected non-nil engine")
	require.False(t, engine.loaded, "expected engine to be unloaded initially")
}

func TestAuthorizeWithoutPolicy(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("allows by default", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{"user": "test"})
		require.NoError(t, err, "unexpected error: %v", err)
		require.True(t, allowed, "expected allowed when no policy loaded")
	})
}

func TestLoadPolicyAndAuthorize(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("load valid policy", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "policy-*.rego")
		require.NoError(t, err)
		_, err = f.WriteString(`package apa.authz

default allow = false

allow {
	input.path == "/admin/health"
}

allow {
	input.user == "admin"
}`)
		require.NoError(t, err)
		f.Close()

		require.NoError(t, engine.LoadPolicy(ctx, f.Name()), "unexpected error loading policy")
		require.True(t, engine.loaded, "expected engine to be loaded after LoadPolicy")
	})

	t.Run("authorized health path", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/health",
			"user": "anonymous",
		})
		require.NoError(t, err, "unexpected error: %v", err)
		require.True(t, allowed, "expected allowed for health path")
	})

	t.Run("authorized admin user", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/secrets",
			"user": "admin",
		})
		require.NoError(t, err, "unexpected error: %v", err)
		require.True(t, allowed, "expected allowed for admin user")
	})

	t.Run("denied unauthorized user", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/secrets",
			"user": "attacker",
		})
		require.NoError(t, err, "unexpected error: %v", err)
		require.False(t, allowed, "expected denied for unauthorized user")
	})
}

func TestLoadPolicyFromInvalidFile(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("missing file", func(t *testing.T) {
		err := engine.LoadPolicy(ctx, "/nonexistent/missing.rego")
		require.Error(t, err, "expected error for missing file")
	})

	t.Run("syntax error in policy", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad-*.rego")
		require.NoError(t, err)
		_, err = f.WriteString("package bad\n\nx = 1 + ")
		require.NoError(t, err)
		f.Close()

		err = engine.LoadPolicy(ctx, f.Name())
		require.Error(t, err, "expected error for syntactically invalid policy")
	})
}

func TestPolicyDecisionFormat(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	f, err := os.CreateTemp(t.TempDir(), "format-*.rego")
	require.NoError(t, err)
	_, err = f.WriteString(`package apa.authz

default allow = "maybe"`)
	require.NoError(t, err)
	f.Close()

	require.NoError(t, engine.LoadPolicy(ctx, f.Name()), "unexpected error: %v", err)

	_, err = engine.Authorize(ctx, map[string]interface{}{"user": "test"})
	require.Error(t, err, "expected error for non-boolean allow value")
}

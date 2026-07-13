package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPolicyEnforcer(t *testing.T) {
	t.Run("valid policy file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "policy.yaml")
		err := os.WriteFile(path, []byte("trusted_authors:\n  - \"test-author\""), 0644)
		require.NoError(t, err)

		p, err := NewPolicyEnforcer(path)
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Len(t, p.config.TrustedAuthors, 1)
		assert.Equal(t, "test-author", p.config.TrustedAuthors[0])
	})

	t.Run("missing policy file", func(t *testing.T) {
		_, err := NewPolicyEnforcer("/nonexistent/policy.yaml")
		assert.Error(t, err)
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		err := os.WriteFile(path, []byte("{{invalid yaml}}"), 0644)
		require.NoError(t, err)

		_, err = NewPolicyEnforcer(path)
		assert.Error(t, err)
	})

	t.Run("empty policy file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yaml")
		err := os.WriteFile(path, []byte(""), 0644)
		require.NoError(t, err)

		p, err := NewPolicyEnforcer(path)
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Empty(t, p.config.TrustedAuthors)
	})

	t.Run("policy with many trusted authors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "policy.yaml")
		content := "trusted_authors:\n"
		for i := range 10 {
			content += "  - \"author-" + string(rune('a'+i)) + "\"\n"
		}
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)

		p, err := NewPolicyEnforcer(path)
		require.NoError(t, err)
		assert.Len(t, p.config.TrustedAuthors, 10)
	})
}

func TestAuthorize(t *testing.T) {
	createEnforcer := func(t *testing.T, authors ...string) *PolicyEnforcerImpl {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "policy.yaml")
		content := "trusted_authors:\n"
		for _, a := range authors {
			content += "  - \"" + a + "\"\n"
		}
		err := os.WriteFile(path, []byte(content), 0644)
		require.NoError(t, err)
		p, err := NewPolicyEnforcer(path)
		require.NoError(t, err)
		return p
	}

	ctx := context.Background()

	t.Run("load_module action allowed", func(t *testing.T) {
		p := createEnforcer(t, "authorized-user")
		allowed, reason, err := p.Authorize(ctx, "any", "load_module", "any-resource")
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, "authorized", reason)
	})

	t.Run("run_module action allowed", func(t *testing.T) {
		p := createEnforcer(t, "authorized-user")
		allowed, reason, err := p.Authorize(ctx, "any", "run_module", "any-resource")
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, "authorized", reason)
	})

	t.Run("unknown action denied", func(t *testing.T) {
		p := createEnforcer(t, "authorized-user")
		allowed, reason, err := p.Authorize(ctx, "any", "delete", "any-resource")
		assert.NoError(t, err)
		assert.False(t, allowed)
		assert.Contains(t, reason, "unauthorized")
	})

	t.Run("empty subject allowed for module actions", func(t *testing.T) {
		p := createEnforcer(t, "authorized-user")
		allowed, reason, err := p.Authorize(ctx, "", "load_module", "module.so")
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, "authorized", reason)
	})

	t.Run("non-module action with empty policy", func(t *testing.T) {
		p := createEnforcer(t)
		allowed, _, err := p.Authorize(ctx, "any", "unknown", "resource")
		assert.NoError(t, err)
		assert.False(t, allowed)
	})
}

func TestPolicyEnforcerInterface(t *testing.T) {
	var _ PolicyEnforcer = (*PolicyEnforcerImpl)(nil)
}

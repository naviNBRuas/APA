package rbac

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRBAC(t *testing.T) *RBACImpl {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRBAC(logger, &Config{PolicyFile: ""})
}

func TestNewRBAC(t *testing.T) {
	rb := newTestRBAC(t)
	require.NotNil(t, rb)
	assert.Empty(t, rb.roles)
	assert.Empty(t, rb.users)
}

func TestRBAC_AddRole(t *testing.T) {
	rb := newTestRBAC(t)

	t.Run("add new role", func(t *testing.T) {
		err := rb.AddRole("admin")
		assert.NoError(t, err)
		_, exists := rb.roles["admin"]
		assert.True(t, exists)
	})

	t.Run("add duplicate role", func(t *testing.T) {
		err := rb.AddRole("admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestRBAC_AddPermission(t *testing.T) {
	rb := newTestRBAC(t)
	require.NoError(t, rb.AddRole("admin"))

	t.Run("add permission to existing role", func(t *testing.T) {
		err := rb.AddPermission("admin", "read", "document")
		assert.NoError(t, err)
		assert.True(t, rb.roles["admin"]["read"]["document"])
	})

	t.Run("add permission to non-existent role", func(t *testing.T) {
		err := rb.AddPermission("nonexistent", "read", "document")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRBAC_AssignRole(t *testing.T) {
	rb := newTestRBAC(t)
	require.NoError(t, rb.AddRole("admin"))

	t.Run("assign role to new user", func(t *testing.T) {
		err := rb.AssignRole("alice", "admin")
		assert.NoError(t, err)
		assert.Contains(t, rb.users["alice"], "admin")
	})

	t.Run("assign duplicate role to same user", func(t *testing.T) {
		err := rb.AssignRole("alice", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already has role")
	})

	t.Run("assign non-existent role", func(t *testing.T) {
		err := rb.AssignRole("bob", "sudoer")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRBAC_Authorize(t *testing.T) {
	ctx := context.Background()

	t.Run("authorized user", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("admin"))
		require.NoError(t, rb.AddPermission("admin", "read", "document"))
		require.NoError(t, rb.AssignRole("alice", "admin"))

		allowed, reason, err := rb.Authorize(ctx, "alice", "read", "document")
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, "authorized", reason)
	})

	t.Run("unauthorized action", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("admin"))
		require.NoError(t, rb.AddPermission("admin", "read", "document"))
		require.NoError(t, rb.AssignRole("alice", "admin"))

		allowed, reason, err := rb.Authorize(ctx, "alice", "write", "document")
		assert.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, "permission denied", reason)
	})

	t.Run("non-existent user", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("admin"))
		require.NoError(t, rb.AddPermission("admin", "read", "document"))

		allowed, reason, err := rb.Authorize(ctx, "bob", "read", "document")
		assert.NoError(t, err)
		assert.False(t, allowed)
		assert.Equal(t, "user not found", reason)
	})

	t.Run("role without permissions", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("viewer"))
		require.NoError(t, rb.AssignRole("charlie", "viewer"))

		allowed, _, err := rb.Authorize(ctx, "charlie", "read", "document")
		assert.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("wildcard resource permission", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("admin"))
		require.NoError(t, rb.AddPermission("admin", "read", "*"))
		require.NoError(t, rb.AssignRole("alice", "admin"))

		allowed, _, err := rb.Authorize(ctx, "alice", "read", "any-resource")
		assert.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("multiple roles - one authorizes", func(t *testing.T) {
		rb := newTestRBAC(t)
		require.NoError(t, rb.AddRole("viewer"))
		require.NoError(t, rb.AddRole("editor"))
		require.NoError(t, rb.AddPermission("viewer", "read", "document"))
		require.NoError(t, rb.AddPermission("editor", "write", "document"))
		require.NoError(t, rb.AssignRole("alice", "viewer"))
		require.NoError(t, rb.AssignRole("alice", "editor"))

		readAllowed, _, _ := rb.Authorize(ctx, "alice", "read", "document")
		writeAllowed, _, _ := rb.Authorize(ctx, "alice", "write", "document")
		deleteAllowed, _, _ := rb.Authorize(ctx, "alice", "delete", "document")

		assert.True(t, readAllowed)
		assert.True(t, writeAllowed)
		assert.False(t, deleteAllowed)
	})
}

func TestRBAC_RevokeRole(t *testing.T) {
	rb := newTestRBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AssignRole("alice", "admin"))

	t.Run("revoke existing role", func(t *testing.T) {
		err := rb.RevokeRole("alice", "admin")
		assert.NoError(t, err)
		assert.NotContains(t, rb.users["alice"], "admin")
	})

	t.Run("revoke already revoked role", func(t *testing.T) {
		err := rb.RevokeRole("alice", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have role")
	})

	t.Run("revoke from non-existent user", func(t *testing.T) {
		err := rb.RevokeRole("bob", "admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRBAC_RemovePermission(t *testing.T) {
	rb := newTestRBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AddPermission("admin", "read", "document"))

	t.Run("remove existing permission", func(t *testing.T) {
		err := rb.RemovePermission("admin", "read", "document")
		assert.NoError(t, err)
		assert.False(t, rb.roles["admin"]["read"]["document"])
	})

	t.Run("remove from non-existent role", func(t *testing.T) {
		err := rb.RemovePermission("sudoer", "read", "document")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("remove non-existent action", func(t *testing.T) {
		err := rb.RemovePermission("admin", "write", "document")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("remove non-existent resource", func(t *testing.T) {
		err := rb.RemovePermission("admin", "read", "spreadsheet")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRBAC_RemoveRole(t *testing.T) {
	rb := newTestRBAC(t)
	require.NoError(t, rb.AddRole("admin"))

	t.Run("remove existing role", func(t *testing.T) {
		err := rb.RemoveRole("admin")
		assert.NoError(t, err)
		_, exists := rb.roles["admin"]
		assert.False(t, exists)
	})

	t.Run("remove non-existent role", func(t *testing.T) {
		err := rb.RemoveRole("admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}

func TestRBAC_FullWorkflow(t *testing.T) {
	rb := newTestRBAC(t)
	ctx := context.Background()

	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AddRole("viewer"))
	require.NoError(t, rb.AddPermission("admin", "read", "document"))
	require.NoError(t, rb.AddPermission("admin", "write", "document"))
	require.NoError(t, rb.AddPermission("viewer", "read", "document"))
	require.NoError(t, rb.AssignRole("alice", "admin"))
	require.NoError(t, rb.AssignRole("bob", "viewer"))

	t.Run("initial authorize", func(t *testing.T) {
		aliceRead, reason, err := rb.Authorize(ctx, "alice", "read", "document")
		assert.NoError(t, err)
		_ = reason
		aliceWrite, reason, err := rb.Authorize(ctx, "alice", "write", "document")
		assert.NoError(t, err)
		_ = reason
		bobRead, reason, err := rb.Authorize(ctx, "bob", "read", "document")
		assert.NoError(t, err)
		_ = reason
		bobWrite, reason, err := rb.Authorize(ctx, "bob", "write", "document")
		assert.NoError(t, err)
		_ = reason

		assert.True(t, aliceRead)
		assert.True(t, aliceWrite)
		assert.True(t, bobRead)
		assert.False(t, bobWrite)
	})

	t.Run("after permission removal", func(t *testing.T) {
		require.NoError(t, rb.RemovePermission("admin", "write", "document"))

		aliceWrite, _, _ := rb.Authorize(ctx, "alice", "write", "document")
		assert.False(t, aliceWrite)
	})

	t.Run("after role revocation", func(t *testing.T) {
		require.NoError(t, rb.RevokeRole("bob", "viewer"))

		bobRead, _, _ := rb.Authorize(ctx, "bob", "read", "document")
		assert.False(t, bobRead)
	})

	t.Run("after role removal", func(t *testing.T) {
		require.NoError(t, rb.RemoveRole("viewer"))

		charlieViewer := rb.AssignRole("charlie", "viewer")
		assert.Error(t, charlieViewer)
	})
}

// ---------------------------------------------------------------------------
// OPA RBAC tests
// ---------------------------------------------------------------------------

func newTestOPARBAC(t *testing.T) *OPARBAC {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	rb, err := NewOPARBAC(logger, &Config{PolicyFile: ""})
	require.NoError(t, err)
	return rb
}

func TestNewOPARBAC(t *testing.T) {
	t.Run("without policy file", func(t *testing.T) {
		rb := newTestOPARBAC(t)
		require.NotNil(t, rb)
		assert.Empty(t, rb.roles)
		assert.Empty(t, rb.users)
	})

	t.Run("with valid policy file", func(t *testing.T) {
		dir := t.TempDir()
		policyPath := filepath.Join(dir, "policy.yaml")
		policyContent := `
roles:
  - name: admin
    permissions:
      - action: read
        resource: document
      - action: write
        resource: document
users:
  - name: alice
    roles:
      - admin
`
		require.NoError(t, os.WriteFile(policyPath, []byte(policyContent), 0644))

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		rb, err := NewOPARBAC(logger, &Config{PolicyFile: policyPath})
		require.NoError(t, err)
		assert.NotNil(t, rb)
		assert.Contains(t, rb.roles, "admin")
		assert.Contains(t, rb.users, "alice")
	})

	t.Run("with invalid policy file path", func(t *testing.T) {
		logger := slog.Default()
		_, err := NewOPARBAC(logger, &Config{PolicyFile: "/nonexistent/policy.yaml"})
		assert.Error(t, err)
	})

	t.Run("with invalid YAML policy", func(t *testing.T) {
		dir := t.TempDir()
		policyPath := filepath.Join(dir, "bad.yaml")
		require.NoError(t, os.WriteFile(policyPath, []byte("{{invalid}}"), 0644))

		logger := slog.Default()
		_, err := NewOPARBAC(logger, &Config{PolicyFile: policyPath})
		assert.Error(t, err)
	})
}

func TestOPARBAC_AddRole(t *testing.T) {
	rb := newTestOPARBAC(t)

	err := rb.AddRole("admin")
	assert.NoError(t, err)

	err = rb.AddRole("admin")
	assert.Error(t, err)
}

func TestOPARBAC_AssignRole(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))

	err := rb.AssignRole("alice", "admin")
	assert.NoError(t, err)
	assert.Contains(t, rb.users["alice"], "admin")

	err = rb.AssignRole("alice", "admin")
	assert.Error(t, err)
}

func TestOPARBAC_Authorize(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AddPermission("admin", "read", "document"))
	require.NoError(t, rb.AssignRole("alice", "admin"))

	ctx := context.Background()

	allowed, _, err := rb.Authorize(ctx, "alice", "read", "document")
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, _, _ = rb.Authorize(ctx, "alice", "write", "document")
	assert.False(t, allowed)

	allowed, _, _ = rb.Authorize(ctx, "bob", "read", "document")
	assert.False(t, allowed)

	allowed, _, _ = rb.Authorize(ctx, "alice", "read", "other")
	assert.False(t, allowed)
}

func TestOPARBAC_WildcardPermission(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AddPermission("admin", "read", "*"))
	require.NoError(t, rb.AssignRole("alice", "admin"))

	ctx := context.Background()

	allowed, reason, err := rb.Authorize(ctx, "alice", "read", "any-resource")
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, "authorized", reason)
}

func TestOPARBAC_RevokeRole(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AssignRole("alice", "admin"))

	err := rb.RevokeRole("alice", "admin")
	assert.NoError(t, err)
	assert.NotContains(t, rb.users["alice"], "admin")

	err = rb.RevokeRole("alice", "admin")
	assert.Error(t, err)
}

func TestOPARBAC_RemoveRole(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))

	err := rb.RemoveRole("admin")
	assert.NoError(t, err)

	err = rb.RemoveRole("admin")
	assert.Error(t, err)
}

func TestOPARBAC_RemovePermission(t *testing.T) {
	rb := newTestOPARBAC(t)
	require.NoError(t, rb.AddRole("admin"))
	require.NoError(t, rb.AddPermission("admin", "read", "document"))

	err := rb.RemovePermission("admin", "read", "document")
	assert.NoError(t, err)

	err = rb.RemovePermission("admin", "read", "document")
	assert.Error(t, err)

	err = rb.RemovePermission("nonexistent", "read", "document")
	assert.Error(t, err)
}

func TestInterfaceImplementations(t *testing.T) {
	var _ RBAC = (*RBACImpl)(nil)
	var _ RBAC = (*OPARBAC)(nil)
}

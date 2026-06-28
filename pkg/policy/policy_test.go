package policy

import (
	"context"
	"os"
	"testing"
)

func TestNewPolicyEnforcer(t *testing.T) {
	t.Run("valid policy file", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "policy-*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("trusted_authors:\n  - \"test-author\""); err != nil {
			t.Fatal(err)
		}
		f.Close() //nolint:errcheck

		p, err := NewPolicyEnforcer(f.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p == nil {
			t.Fatal("expected non-nil enforcer")
		}
		if len(p.config.TrustedAuthors) != 1 || p.config.TrustedAuthors[0] != "test-author" {
			t.Fatalf("unexpected trusted authors: %v", p.config.TrustedAuthors)
		}
	})

	t.Run("missing policy file", func(t *testing.T) {
		_, err := NewPolicyEnforcer("/nonexistent/policy.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad-*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("{{invalid yaml}}"); err != nil {
			t.Fatal(err)
		}
		f.Close() //nolint:errcheck

		_, err = NewPolicyEnforcer(f.Name())
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

func TestPolicyEnforcerImpl_Authorize(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "policy-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("trusted_authors:\n  - \"test-author\""); err != nil {
		t.Fatal(err)
	}
	f.Close() //nolint:errcheck

	p, err := NewPolicyEnforcer(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()

	t.Run("load_module action", func(t *testing.T) {
		allowed, reason, err := p.Authorize(ctx, "any-subject", "load_module", "any-resource")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("expected allowed, got denied: %s", reason)
		}
	})

	t.Run("run_module action", func(t *testing.T) {
		allowed, reason, err := p.Authorize(ctx, "any-subject", "run_module", "any-resource")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("expected allowed, got denied: %s", reason)
		}
	})

	t.Run("unknown action", func(t *testing.T) {
		allowed, reason, err := p.Authorize(ctx, "any-subject", "unknown_action", "any-resource")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("expected denied for unknown action")
		}
		if reason == "" {
			t.Fatal("expected non-empty reason")
		}
	})
}

func TestPolicyEnforcerInterface(t *testing.T) {
	var _ PolicyEnforcer = (*PolicyEnforcerImpl)(nil)
}

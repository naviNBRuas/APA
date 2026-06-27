package opa

import (
	"context"
	"os"
	"testing"
)

func TestNewOPAPolicyEngine(t *testing.T) {
	engine := NewOPAPolicyEngine()
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
	if engine.loaded {
		t.Fatal("expected engine to be unloaded initially")
	}
}

func TestAuthorizeWithoutPolicy(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("allows by default", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{"user": "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed when no policy loaded")
		}
	})
}

func TestLoadPolicyAndAuthorize(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("load valid policy", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "policy-*.rego")
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString(`package apa.authz

default allow = false

allow {
	input.path == "/admin/health"
}

allow {
	input.user == "admin"
}`)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		if err := engine.LoadPolicy(ctx, f.Name()); err != nil {
			t.Fatalf("unexpected error loading policy: %v", err)
		}
		if !engine.loaded {
			t.Fatal("expected engine to be loaded after LoadPolicy")
		}
	})

	t.Run("authorized health path", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/health",
			"user": "anonymous",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed for health path")
		}
	})

	t.Run("authorized admin user", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/secrets",
			"user": "admin",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("expected allowed for admin user")
		}
	})

	t.Run("denied unauthorized user", func(t *testing.T) {
		allowed, err := engine.Authorize(ctx, map[string]interface{}{
			"path": "/admin/secrets",
			"user": "attacker",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("expected denied for unauthorized user")
		}
	})
}

func TestLoadPolicyFromInvalidFile(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	t.Run("missing file", func(t *testing.T) {
		err := engine.LoadPolicy(ctx, "/nonexistent/missing.rego")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("syntax error in policy", func(t *testing.T) {
		f, err := os.CreateTemp(t.TempDir(), "bad-*.rego")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("package bad\n\nx = 1 + "); err != nil {
			t.Fatal(err)
		}
		f.Close()

		err = engine.LoadPolicy(ctx, f.Name())
		if err == nil {
			t.Fatal("expected error for syntactically invalid policy")
		}
	})
}

func TestPolicyDecisionFormat(t *testing.T) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	f, err := os.CreateTemp(t.TempDir(), "format-*.rego")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(`package apa.authz

default allow = "maybe"`)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	if err := engine.LoadPolicy(ctx, f.Name()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = engine.Authorize(ctx, map[string]interface{}{"user": "test"})
	if err == nil {
		t.Fatal("expected error for non-boolean allow value")
	}
}

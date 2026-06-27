package opa

import (
	"context"
	"os"
	"testing"
)

const benchPolicy = `package apa.authz

default allow = false

allow {
	input.user == "admin"
}`

func BenchmarkAuthorizeRequest(b *testing.B) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	f, err := os.CreateTemp(b.TempDir(), "policy-*.rego")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.WriteString(benchPolicy); err != nil {
		b.Fatal(err)
	}
	f.Close()

	if err := engine.LoadPolicy(ctx, f.Name()); err != nil {
		b.Fatal(err)
	}

	input := map[string]interface{}{
		"user": "admin",
		"path": "/admin/secrets",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Authorize(ctx, input)
	}
}

func BenchmarkAuthorizeParallel(b *testing.B) {
	engine := NewOPAPolicyEngine()
	ctx := context.Background()

	f, err := os.CreateTemp(b.TempDir(), "policy-*.rego")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.WriteString(benchPolicy); err != nil {
		b.Fatal(err)
	}
	f.Close()

	if err := engine.LoadPolicy(ctx, f.Name()); err != nil {
		b.Fatal(err)
	}

	input := map[string]interface{}{
		"user": "admin",
		"path": "/admin/secrets",
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			engine.Authorize(ctx, input)
		}
	})
}

func BenchmarkPolicyLoad(b *testing.B) {
	ctx := context.Background()

	f, err := os.CreateTemp(b.TempDir(), "policy-*.rego")
	if err != nil {
		b.Fatal(err)
	}
	if _, err := f.WriteString(benchPolicy); err != nil {
		b.Fatal(err)
	}
	f.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := NewOPAPolicyEngine()
		engine.LoadPolicy(ctx, f.Name())
	}
}

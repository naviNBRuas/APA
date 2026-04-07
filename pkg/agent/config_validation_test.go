package agent

import "testing"

func TestValidateConfig(t *testing.T) {
	cfg := &Config{}
	if err := validateConfig(cfg); err == nil {
		t.Fatalf("expected error for empty config")
	}

	cfg.AdminListenAddress = ":8080"
	cfg.ModulePath = "modules"
	cfg.IdentityFilePath = "id"
	cfg.PolicyPath = "policy"
	cfg.ControllerPath = "controllers"

	if err := validateConfig(cfg); err != nil {
		t.Fatalf("unexpected error after populating required fields: %v", err)
	}

	cfg.AdminTLSRequireClientCert = true
	cfg.AdminTLSClientCA = ""
	if err := validateConfig(cfg); err == nil {
		t.Fatalf("expected error when client cert required without CA")
	}
}

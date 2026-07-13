package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	cfg := &Config{}
	err := validateConfig(cfg)
	require.Error(t, err, "expected error for empty config")

	cfg.AdminListenAddress = ":8080"
	cfg.ModulePath = "modules"
	cfg.IdentityFilePath = "id"
	cfg.PolicyPath = "policy"
	cfg.ControllerPath = "controllers"

	err = validateConfig(cfg)
	require.NoError(t, err, "unexpected error after populating required fields")

	cfg.AdminTLSRequireClientCert = true
	cfg.AdminTLSClientCA = ""
	err = validateConfig(cfg)
	require.Error(t, err, "expected error when client cert required without CA")
}

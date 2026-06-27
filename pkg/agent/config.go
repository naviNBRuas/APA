package agent

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func validateConfig(c *Config) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.AdminListenAddress == "" {
		return fmt.Errorf("admin_listen_address is required")
	}
	if c.ModulePath == "" {
		return fmt.Errorf("module_path is required")
	}
	if c.IdentityFilePath == "" {
		return fmt.Errorf("identity_file_path is required")
	}
	if c.PolicyPath == "" {
		return fmt.Errorf("policy_path is required")
	}
	if c.ControllerPath == "" {
		return fmt.Errorf("controller_path is required")
	}
	if c.AdminTLSRequireClientCert && c.AdminTLSClientCA == "" {
		return fmt.Errorf("admin_tls_client_ca is required when admin_tls_require_client_cert is true")
	}
	return nil
}

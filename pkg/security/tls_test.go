package security

import (
	"crypto/tls"
	"testing"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	certPEM, keyPEM, err := GenerateSelfSignedCert("test.example.com", 365)
	if err != nil {
		t.Fatalf("Failed to generate self-signed certificate: %v", err)
	}

	if len(certPEM) == 0 {
		t.Error("Generated certificate is empty")
	}

	if len(keyPEM) == 0 {
		t.Error("Generated key is empty")
	}
}

func TestCreateTLSConfig(t *testing.T) {
	config := TLSConfig{
		ServerName: "test.example.com",
		ClientAuth: tls.NoClientCert,
	}

	tlsConfig, err := CreateTLSConfig(config)
	if err != nil {
		t.Fatalf("Failed to create TLS config: %v", err)
	}

	if tlsConfig.ServerName != "test.example.com" {
		t.Errorf("Expected ServerName 'test.example.com', got '%s'", tlsConfig.ServerName)
	}

	if tlsConfig.ClientAuth != tls.NoClientCert {
		t.Errorf("Expected ClientAuth tls.NoClientCert, got %v", tlsConfig.ClientAuth)
	}
}
package security

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// TLSConfig holds the configuration for TLS connections
type TLSConfig struct {
	CertFile       string
	KeyFile        string
	CAFile         string
	ClientAuth     tls.ClientAuthType
	ServerName     string
	InsecureSkipVerify bool
}

// GenerateSelfSignedCert generates a self-signed certificate and key for testing
func GenerateSelfSignedCert(commonName string, validityDays int) ([]byte, []byte, error) {
	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"APA Project"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(validityDays) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	privKeyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyDER,
	})

	return certPEM, keyPEM, nil
}

// LoadTLSCertificate loads a TLS certificate from PEM files
func LoadTLSCertificate(certFile, keyFile string) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to load TLS certificate: %w", err)
	}
	return cert, nil
}

// CreateTLSConfig creates a TLS configuration for the agent
func CreateTLSConfig(config TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		ServerName:         config.ServerName,
		InsecureSkipVerify: config.InsecureSkipVerify,
		ClientAuth:         config.ClientAuth,
	}

	// Load certificate if provided
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := LoadTLSCertificate(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if config.CAFile != "" {
		caCertPool, err := LoadCACertificates(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load CA certificates: %w", err)
		}
		tlsConfig.RootCAs = caCertPool
		if config.ClientAuth != tls.NoClientCert {
			tlsConfig.ClientCAs = caCertPool
		}
	}

	return tlsConfig, nil
}

// LoadCACertificates loads CA certificates from a PEM file
func LoadCACertificates(caFile string) (*x509.CertPool, error) {
	// Implementation would load CA certificates from file
	// For now, we'll return an empty pool
	pool := x509.NewCertPool()
	return pool, nil
}
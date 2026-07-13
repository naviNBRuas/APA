package security

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	tests := []struct {
		name         string
		commonName   string
		validityDays int
	}{
		{"standard cert", "test.example.com", 365},
		{"short validity", "localhost", 1},
		{"empty CN", "", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPEM, keyPEM, err := GenerateSelfSignedCert(tt.commonName, tt.validityDays)
			require.NoError(t, err)
			require.NotEmpty(t, certPEM, "certificate PEM should not be empty")
			require.NotEmpty(t, keyPEM, "key PEM should not be empty")

			block, _ := pem.Decode(certPEM)
			require.NotNil(t, block)
			assert.Equal(t, "CERTIFICATE", block.Type)

			cert, err := x509.ParseCertificate(block.Bytes)
			require.NoError(t, err)
			assert.Equal(t, tt.commonName, cert.Subject.CommonName)
			assert.True(t, cert.IsCA)

			block, _ = pem.Decode(keyPEM)
			require.NotNil(t, block)
			assert.Equal(t, "EC PRIVATE KEY", block.Type)
		})
	}
}

func TestLoadTLSCertificate(t *testing.T) {
	certPEM, keyPEM, err := GenerateSelfSignedCert("test.example.com", 365)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	err = os.WriteFile(certFile, certPEM, 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyFile, keyPEM, 0644)
	require.NoError(t, err)

	t.Run("valid cert and key", func(t *testing.T) {
		cert, err := LoadTLSCertificate(certFile, keyFile)
		require.NoError(t, err)
		require.NotNil(t, cert)
		assert.NotEmpty(t, cert.Certificate)
	})

	t.Run("missing cert file", func(t *testing.T) {
		_, err := LoadTLSCertificate("/nonexistent/cert.pem", keyFile)
		assert.Error(t, err)
	})

	t.Run("missing key file", func(t *testing.T) {
		_, err := LoadTLSCertificate(certFile, "/nonexistent/key.pem")
		assert.Error(t, err)
	})
}

func TestCreateTLSConfig(t *testing.T) {
	t.Run("basic config", func(t *testing.T) {
		cfg := TLSConfig{
			ServerName: "test.example.com",
			ClientAuth: tls.NoClientCert,
		}

		tlsCfg, err := CreateTLSConfig(cfg)
		require.NoError(t, err)
		require.NotNil(t, tlsCfg)

		assert.Equal(t, "test.example.com", tlsCfg.ServerName)
		assert.Equal(t, tls.NoClientCert, tlsCfg.ClientAuth)
		assert.False(t, tlsCfg.InsecureSkipVerify)
	})

	t.Run("with insecure skip verify", func(t *testing.T) {
		cfg := TLSConfig{
			ServerName:         "test.example.com",
			InsecureSkipVerify: true,
		}

		tlsCfg, err := CreateTLSConfig(cfg)
		require.NoError(t, err)
		assert.True(t, tlsCfg.InsecureSkipVerify)
	})

	t.Run("with client cert and key", func(t *testing.T) {
		certPEM, keyPEM, err := GenerateSelfSignedCert("test.example.com", 365)
		require.NoError(t, err)

		dir := t.TempDir()
		certFile := filepath.Join(dir, "cert.pem")
		keyFile := filepath.Join(dir, "key.pem")

		err = os.WriteFile(certFile, certPEM, 0644)
		require.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0644)
		require.NoError(t, err)

		cfg := TLSConfig{
			CertFile:   certFile,
			KeyFile:    keyFile,
			ServerName: "test.example.com",
		}

		tlsCfg, err := CreateTLSConfig(cfg)
		require.NoError(t, err)
		assert.Len(t, tlsCfg.Certificates, 1)
	})

	t.Run("with require client cert", func(t *testing.T) {
		cfg := TLSConfig{
			ServerName: "test.example.com",
			ClientAuth: tls.RequireAndVerifyClientCert,
		}

		tlsCfg, err := CreateTLSConfig(cfg)
		require.NoError(t, err)
		assert.Equal(t, tls.RequireAndVerifyClientCert, tlsCfg.ClientAuth)
	})
}

func TestLoadCACertificates(t *testing.T) {
	pool, err := LoadCACertificates("/nonexistent/ca.pem")
	require.NoError(t, err)
	require.NotNil(t, pool)
}

func TestGenerateSelfSignedCert_VerifyCertDetails(t *testing.T) {
	certPEM, _, err := GenerateSelfSignedCert("test-server.local", 30)
	require.NoError(t, err)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	assert.Equal(t, "test-server.local", cert.Subject.CommonName)
	assert.Equal(t, []string{"APA Project"}, cert.Subject.Organization)
	assert.True(t, cert.IsCA)

	assert.True(t, cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0)
	assert.True(t, cert.KeyUsage&x509.KeyUsageDigitalSignature != 0)
}

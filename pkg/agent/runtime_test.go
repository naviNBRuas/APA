package agent

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func generateTestKeys(t *testing.T) (string, string) {
	priv, pub, err := crypto.GenerateEd25519Key(nil)
	require.NoError(t, err)

	// Create a dummy identity file
	identityFile, err := os.CreateTemp("", "agent-identity-*.json")
	require.NoError(t, err)

	privBytes, err := crypto.MarshalPrivateKey(priv)
	require.NoError(t, err)

	identityData := &identityState{
		PrivateKey: hex.EncodeToString(privBytes),
	}

	data, err := json.Marshal(identityData)
	require.NoError(t, err)

	_, err = identityFile.Write(data)
	require.NoError(t, err)
	identityFile.Close()

	raw, err := pub.Raw()
	require.NoError(t, err)

	return hex.EncodeToString(raw), identityFile.Name()
}

func TestNewRuntime(t *testing.T) {
	pubKey, identityPath := generateTestKeys(t)
	defer os.Remove(identityPath)

	// Create a dummy policy file
	policyFile, err := os.CreateTemp("", "policy-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(policyFile.Name())
	_, err = policyFile.WriteString(`
trusted_authors:
  - "naviNBRuas"
`)
	assert.NoError(t, err)
	policyFile.Close()

	// Create a dummy signing private key file
	signingPrivKeyFile, err := os.CreateTemp("", "signing_private-*.key")
	assert.NoError(t, err)
	defer os.Remove(signingPrivKeyFile.Name())
	_, err = signingPrivKeyFile.WriteString("66de82dd3b2ab364a58c741e152f1bdb195cddde0e7e30431c466ea647ea733b1041e474e8c1fc4cf42a183dcfc27ccefc0fde534368f3aff3ee9856a8a960c4")
	assert.NoError(t, err)
	signingPrivKeyFile.Close()

	// Create a dummy config file
	config := `
admin_listen_address: ":8080"
log_level: "debug"
module_path: "/tmp/modules"
identity_file_path: "` + identityPath + `"
policy_path: "` + policyFile.Name() + `"
signing_priv_key_path: "` + signingPrivKeyFile.Name() + `"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    []
  heartbeat_interval: "10s"
  service_tag: "apa-test"
update:
  public_key: "` + pubKey + `"
`
	tmpfile, err := os.CreateTemp("", "agent-config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.WriteString(config)
	assert.NoError(t, err)
	tmpfile.Close()

	// Test NewRuntime with a valid config
	rt, err := NewRuntime(tmpfile.Name(), "v0.1.0")
	assert.NoError(t, err)
	assert.NotNil(t, rt)
	assert.NotNil(t, rt.logger)
	assert.NotNil(t, rt.identity)
	assert.NotNil(t, rt.moduleManager)
	assert.NotNil(t, rt.p2p)
	assert.NotNil(t, rt.updateManager)
	assert.NotNil(t, rt.healthController)
	assert.NotNil(t, rt.recoveryController)

	// Test NewRuntime with a non-existent config
	_, err = NewRuntime("non-existent-config.yaml", "v0.1.0")
	assert.Error(t, err)
}

func TestHealthEndpoint(t *testing.T) {
	pubKey, identityPath := generateTestKeys(t)
	defer os.Remove(identityPath)

	// Create a dummy policy file
	policyFile, err := os.CreateTemp("", "policy-*.yaml")
	require.NoError(t, err)
	defer os.Remove(policyFile.Name())
	_, err = policyFile.WriteString(`
trusted_authors:
  - "naviNBRuas"
`)
	require.NoError(t, err)
	policyFile.Close()

	// Create a temporary config file
	configFile, err := os.CreateTemp("", "agent-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	// Create a dummy signing private key file
	signingPrivKeyFile, err := os.CreateTemp("", "signing_private-*.key")
	require.NoError(t, err)
	defer os.Remove(signingPrivKeyFile.Name())
	_, err = signingPrivKeyFile.WriteString("66de82dd3b2ab364a58c741e152f1bdb195cddde0e7e30431c466ea647ea733b1041e474e8c1fc4cf42a183dcfc27ccefc0fde534368f3aff3ee9856a8a960c4")
	require.NoError(t, err)
	signingPrivKeyFile.Close()

	_, err = configFile.WriteString(`
admin_listen_address: ":8081"
log_level: "debug"
identity_file_path: "` + identityPath + `"
module_path: "/tmp/modules"
policy_path: "` + policyFile.Name() + `"
signing_priv_key_path: "` + signingPrivKeyFile.Name() + `"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    []
  heartbeat_interval: "10s"
  service_tag: "test-agent"
update:
  public_key: "` + pubKey + `"
`)
	require.NoError(t, err)
	configFile.Close()

	// Create a runtime with the test config
	rt, err := NewRuntime(configFile.Name(), "v0.1.0")
	require.NoError(t, err)

	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(rt.healthHandler))
	defer ts.Close()

	// Make a request to the health endpoint
	res, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer res.Body.Close()

	// Check the response
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", res.Header.Get("Content-Type"))
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, "OK\n", string(body))
}

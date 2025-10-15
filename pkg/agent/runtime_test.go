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

type identityState struct {
	PrivateKey string `json:"private_key"`
}

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

	pubBytes, err := crypto.MarshalPublicKey(pub)
	require.NoError(t, err)

	return hex.EncodeToString(pubBytes), identityFile.Name()
}

func TestNewRuntime(t *testing.T) {
	pubKey, identityPath := generateTestKeys(t)
	defer os.Remove(identityPath)

	// Create a dummy config file
	config := `
admin_listen_address: ":8080"
log_level: "debug"
module_path: "/tmp/modules"
identity_path: "` + identityPath + `"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    []
  heartbeat_interval: "10s"
  service_tag: "test-agent"
update:
  enabled: true
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

	// Test NewRuntime with a non-existent config
	_, err = NewRuntime("non-existent-config.yaml", "v0.1.0")
	assert.Error(t, err)
}

func TestHealthEndpoint(t *testing.T) {
	pubKey, identityPath := generateTestKeys(t)
	defer os.Remove(identityPath)

	// Create a temporary config file
	configFile, err := os.CreateTemp("", "agent-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(`
admin_listen_address: ":8081"
log_level: "debug"
identity_path: "` + identityPath + `"
module_path: "/tmp/modules"
p2p:
  listen_addresses:
    - "/ip4/0.0.0.0/tcp/0"
  bootstrap_peers:
    []
  heartbeat_interval: "10s"
  service_tag: "test-agent"
update:
  enabled: true
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
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, "OK\n", string(body))
}

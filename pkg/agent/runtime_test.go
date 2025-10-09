package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	// Create a temporary config file
	configFile, err := os.CreateTemp("", "agent-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(`
admin_listen_address: ":8081"
log_level: "debug"
`)
	require.NoError(t, err)
	configFile.Close()

	// Create a runtime with the test config
	rt, err := NewRuntime(configFile.Name())
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
	assert.Equal(t, "apa: ok", string(body))
}

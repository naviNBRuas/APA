package agent

import (
	"encoding/json"
	"github.com/naviNBRuas/APA/pkg/networking"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetricsEndpoint(t *testing.T) {
	// Minimal runtime with metrics handler
	rt := &Runtime{}
	rt.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	rt.config = &Config{
		P2P: networking.Config{HeartbeatInterval: time.Second},
	}
	rt.adminPolicyEngine = nil
	rt.startTime = time.Now()
	rt.rateLimiters = make(map[string]*rate.Limiter)
	ts := httptest.NewServer(http.HandlerFunc(rt.metricsHandler))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&metrics))
	_, ok := metrics["uptime_seconds"]
	require.True(t, ok)
	// topics_health is only present if p2p is not nil
	_, _ = metrics["topics_health"]
	// Accept either presence or absence for this minimal test
}

func TestAuditEndpoint_NoLogger(t *testing.T) {
	rt := &Runtime{}
	rt.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	rt.config = &Config{
		P2P: networking.Config{HeartbeatInterval: time.Second},
	}
	rt.rateLimiters = make(map[string]*rate.Limiter)
	ts := httptest.NewServer(http.HandlerFunc(rt.auditHandler))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

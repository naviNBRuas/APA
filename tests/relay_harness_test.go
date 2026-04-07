package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDockerRelayHarness(t *testing.T) {
	if os.Getenv("RUN_DOCKER_P2P") == "" {
		t.Skip("set RUN_DOCKER_P2P=1 to run dockerized relay harness test")
	}

	composePath := filepath.Join("harness", "p2p", "docker-compose.yml")
	absPath, err := filepath.Abs(composePath)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}

	upCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	runCompose(t, upCtx, absPath, "up", "-d", "--build", "--remove-orphans")
	t.Cleanup(func() {
		downCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		runCompose(t, downCtx, absPath, "down", "-v")
	})

	waitHTTP(t, "http://localhost:18080/health", 2*time.Minute)
	waitHTTP(t, "http://localhost:18081/health", 2*time.Minute)
	waitHTTP(t, "http://localhost:18082/health", 2*time.Minute)

	relay := fetchInfo(t, "http://localhost:18080/info")
	nodeA := fetchInfo(t, "http://localhost:18081/info")
	nodeB := fetchInfo(t, "http://localhost:18082/info")
	if nodeB.PeerID == "" {
		t.Fatalf("node_b peer id missing")
	}

	relayForPublic := pickAddr(relay.Addrs, "172.28.")
	if relayForPublic == "" && len(relay.Addrs) > 0 {
		relayForPublic = relay.Addrs[0]
	}

	connectReq := map[string]string{
		"peer_id":    nodeA.PeerID,
		"relay_addr": relayForPublic,
	}
	postJSON(t, "http://localhost:18082/connect", connectReq, http.StatusNoContent)

	// Publish a controller message from node_a -> node_b via relay path
	postJSON(t, "http://localhost:18081/publish", map[string]any{
		"type": "relay-test",
		"data": map[string]any{"message": "hello via relay"},
	}, http.StatusAccepted)

	waitForMessage(t, "http://localhost:18082/messages", "relay-test", 45*time.Second)

	// DHT propagation: node_a puts, node_b fetches
	key := "/apa/test/key"
	postJSON(t, "http://localhost:18081/put-dht", map[string]string{"key": key, "value": "dht-value"}, http.StatusNoContent)
	waitForDHT(t, "http://localhost:18082/get-dht?key="+key, "dht-value", 45*time.Second)
}

type infoPayload struct {
	PeerID string   `json:"peer_id"`
	Addrs  []string `json:"addrs"`
}

type messagePayload struct {
	Type string `json:"type"`
}

type dhtPayload struct {
	Value string `json:"value"`
}

func runCompose(t *testing.T, ctx context.Context, composePath string, args ...string) {
	t.Helper()
	full := append([]string{"compose", "-f", composePath}, args...)
	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Dir = filepath.Dir(composePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose %v failed: %v\n%s", args, err, string(out))
	}
}

func waitHTTP(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url) // #nosec G107
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", url)
		}
		time.Sleep(2 * time.Second)
	}
}

func fetchInfo(t *testing.T, url string) infoPayload {
	t.Helper()
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		t.Fatalf("get info: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var payload infoPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("parse info: %v body=%s", err, string(body))
	}
	return payload
}

func pickAddr(addrs []string, contains string) string {
	for _, a := range addrs {
		if strings.Contains(a, contains) {
			return a
		}
	}
	return ""
}

func postJSON(t *testing.T, url string, body any, expect int) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data)) // #nosec G107
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expect {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d for %s: %s", resp.StatusCode, url, string(b))
	}
}

func waitForMessage(t *testing.T, url string, msgType string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url) // #nosec G107
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			var msgs []messagePayload
			if err := json.Unmarshal(body, &msgs); err == nil {
				for _, m := range msgs {
					if m.Type == msgType {
						return
					}
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("message %s not observed", msgType)
		}
		time.Sleep(2 * time.Second)
	}
}

func waitForDHT(t *testing.T, url string, expect string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		resp, err := http.Get(url) // #nosec G107
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			var payload dhtPayload
			if err := json.Unmarshal(body, &payload); err == nil && payload.Value == expect {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("dht value not observed for %s", url)
		}
		time.Sleep(2 * time.Second)
	}
}

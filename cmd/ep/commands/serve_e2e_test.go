package commands

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServeE2E(t *testing.T) {
	ep := buildBinary(t)
	tmpDir := t.TempDir()
	epHome := filepath.Join(tmpDir, ".ep")
	os.MkdirAll(epHome, 0755)

	initCmd := exec.Command(ep, "init", tmpDir)
	initCmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	initCmd.CombinedOutput()

	// Start serve in background with a unique port
	port := 18080 + int(time.Now().UnixNano()%1000)
	cmd := exec.Command(ep, "serve", "--port", strconv.Itoa(port), "--no-browser")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "EP_HOME="+epHome)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	require.NoError(t, err, "ep serve should start")
	t.Logf("Started ep serve on port %d, PID: %d", port, cmd.Process.Pid)

	// Wait for server to be ready by polling stdout for startup message
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)
	client := &http.Client{Timeout: 5 * time.Second}

	require.True(t, waitForServer(client, baseURL, 15*time.Second), "server should be ready")

	// Test GET /
	t.Log("Testing GET /")
	resp, err := client.Get(baseURL + "/")
	require.NoError(t, err, "GET / should succeed")
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET / should return 200")
	resp.Body.Close()

	// Test GET /assets/app.js (falls back to index.html if not found)
	t.Log("Testing GET /assets/app.js")
	resp, err = client.Get(baseURL + "/assets/app.js")
	require.NoError(t, err, "GET /assets/app.js should succeed")
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /assets/app.js should return 200")
	resp.Body.Close()

	// Test GET /healthz
	t.Log("Testing GET /healthz")
	resp, err = client.Get(baseURL + "/healthz")
	require.NoError(t, err, "GET /healthz should succeed")
	require.Equal(t, http.StatusOK, resp.StatusCode, "GET /healthz should return 200")
	resp.Body.Close()

	// Graceful shutdown via SIGINT
	t.Log("Sending SIGINT for graceful shutdown")
	err = cmd.Process.Signal(os.Interrupt)
	require.NoError(t, err, "signal should be sent")

	// Wait for process to exit (SIGINT causes immediate exit in serve.go)
	err = cmd.Wait()
	t.Logf("Process exited with: %v", err)
	// Exit due to signal is expected and acceptable
	require.True(t, err == nil || strings.Contains(err.Error(), "signal: interrupt"),
		"process should exit cleanly or with signal error, got: %v", err)
}

// waitForServer polls the /healthz endpoint until it returns 200 or timeout.
func waitForServer(client *http.Client, baseURL string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			resp, err := client.Get(baseURL + "/healthz")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/schigh/health/v2/discovery"
)

// kubectl runs a kubectl command and returns stdout.
func kubectl(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// portForward starts a port-forward and returns the local URL and a cleanup func.
func portForward(t *testing.T, svc string, remotePort int) (string, func()) {
	t.Helper()
	localPort := 30000 + remotePort // avoid collisions
	cmd := exec.Command("kubectl", "port-forward", fmt.Sprintf("svc/%s", svc), fmt.Sprintf("%d:%d", localPort, remotePort))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("port-forward %s failed: %v", svc, err)
	}
	// wait for port-forward to be ready
	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	cleanup := func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
	return url, cleanup
}

func httpGet(t *testing.T, url string) (int, []byte) {
	t.Helper()
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func TestMain(m *testing.M) {
	// Verify cluster is accessible
	cmd := exec.Command("kubectl", "cluster-info")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "No Kubernetes cluster accessible. Run 'make e2e-cluster e2e-build e2e-deploy' first.\n")
		os.Exit(1)
	}

	// Verify all pods are ready
	for _, app := range []string{"gateway", "orders", "payments", "postgres", "redis"} {
		cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "app="+app, "--timeout=120s")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Pod %s not ready: %s\n%s\n", app, err, out)
			os.Exit(1)
		}
	}

	os.Exit(m.Run())
}

func TestProbesHealthy(t *testing.T) {
	services := []struct {
		name string
		port int
	}{
		{"gateway-svc", 8181},
		{"orders-svc", 8182},
		{"payments-svc", 8183},
	}

	for _, svc := range services {
		t.Run(svc.name, func(t *testing.T) {
			url, cleanup := portForward(t, svc.name, svc.port)
			defer cleanup()

			for _, path := range []string{"/health/live", "/health/ready", "/health/startup"} {
				code, _ := httpGet(t, url+path)
				if code != 200 {
					t.Errorf("%s%s: expected 200, got %d", svc.name, path, code)
				}
			}
		})
	}
}

func TestSelfDescribingJSON(t *testing.T) {
	url, cleanup := portForward(t, "orders-svc", 8182)
	defer cleanup()

	code, body := httpGet(t, url+"/health/ready")
	if code != 200 {
		t.Fatalf("orders /health/ready: expected 200, got %d", code)
	}

	var checks map[string]json.RawMessage
	if err := json.Unmarshal(body, &checks); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// verify postgres check has metadata
	pgRaw, ok := checks["postgres"]
	if !ok {
		t.Fatal("postgres check not found in response")
	}

	var pg struct {
		Group         string `json:"group"`
		ComponentType string `json:"componentType"`
		Duration      string `json:"duration"`
		LastCheck     string `json:"lastCheck"`
	}
	if err := json.Unmarshal(pgRaw, &pg); err != nil {
		t.Fatalf("unmarshal postgres: %v", err)
	}

	if pg.Group != "database" {
		t.Errorf("expected group 'database', got %q", pg.Group)
	}
	if pg.ComponentType != "datastore" {
		t.Errorf("expected componentType 'datastore', got %q", pg.ComponentType)
	}
	if pg.Duration == "" {
		t.Error("expected non-empty duration")
	}
	if pg.LastCheck == "" {
		t.Error("expected non-empty lastCheck")
	}
}

func TestDiscoveryManifest(t *testing.T) {
	url, cleanup := portForward(t, "gateway-svc", 8181)
	defer cleanup()

	code, body := httpGet(t, url+"/.well-known/health")
	if code != 200 {
		t.Fatalf("gateway /.well-known/health: expected 200, got %d", code)
	}

	var manifest discovery.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if manifest.Service != "gateway" {
		t.Errorf("expected service 'gateway', got %q", manifest.Service)
	}
	if manifest.Status != "pass" {
		t.Errorf("expected status 'pass', got %q", manifest.Status)
	}

	// find the orders-api check and verify DependsOn
	found := false
	for _, check := range manifest.Checks {
		if check.Name == "orders-api" {
			found = true
			if len(check.DependsOn) == 0 {
				t.Error("expected orders-api check to have dependsOn")
			}
			break
		}
	}
	if !found {
		t.Error("orders-api check not found in manifest")
	}
}

func TestDiscoveryGraph(t *testing.T) {
	url, cleanup := portForward(t, "gateway-svc", 8181)
	defer cleanup()

	// We can't use DiscoverGraph directly because the DependsOn URLs
	// use cluster-internal addresses. Instead, verify the manifest chain
	// manually by fetching each service's manifest.

	// Fetch gateway manifest
	_, gwBody := httpGet(t, url+"/.well-known/health")
	var gwManifest discovery.Manifest
	json.Unmarshal(gwBody, &gwManifest)

	if gwManifest.Service != "gateway" {
		t.Fatalf("expected gateway, got %q", gwManifest.Service)
	}

	// Fetch orders manifest
	ordersURL, ordersCleanup := portForward(t, "orders-svc", 8182)
	defer ordersCleanup()

	_, ordersBody := httpGet(t, ordersURL+"/.well-known/health")
	var ordersManifest discovery.Manifest
	json.Unmarshal(ordersBody, &ordersManifest)

	if ordersManifest.Service != "orders" {
		t.Fatalf("expected orders, got %q", ordersManifest.Service)
	}

	// Verify orders has dependency on payments
	hasDep := false
	for _, check := range ordersManifest.Checks {
		for _, dep := range check.DependsOn {
			if strings.Contains(dep, "payments") {
				hasDep = true
			}
		}
	}
	if !hasDep {
		t.Error("orders manifest should have a dependency on payments")
	}

	// Fetch payments manifest
	paymentsURL, paymentsCleanup := portForward(t, "payments-svc", 8183)
	defer paymentsCleanup()

	_, paymentsBody := httpGet(t, paymentsURL+"/.well-known/health")
	var paymentsManifest discovery.Manifest
	json.Unmarshal(paymentsBody, &paymentsManifest)

	if paymentsManifest.Service != "payments" {
		t.Fatalf("expected payments, got %q", paymentsManifest.Service)
	}

	t.Logf("Discovery chain verified: gateway → orders → payments")
	t.Logf("Gateway checks: %d, Orders checks: %d, Payments checks: %d",
		len(gwManifest.Checks), len(ordersManifest.Checks), len(paymentsManifest.Checks))
}

func TestFailureAndRecovery(t *testing.T) {
	ordersURL, cleanup := portForward(t, "orders-svc", 8182)
	defer cleanup()

	// Verify healthy first
	code, _ := httpGet(t, ordersURL+"/health/ready")
	if code != 200 {
		t.Fatalf("orders should be ready initially, got %d", code)
	}

	// Kill Redis
	t.Log("Deleting Redis pod...")
	kubectl(t, "delete", "pod", "-l", "app=redis", "--grace-period=0", "--force")

	// Wait for orders to detect the failure (check interval is 5s)
	t.Log("Waiting for orders to detect Redis failure...")
	deadline := time.Now().Add(30 * time.Second)
	detected := false
	for time.Now().Before(deadline) {
		code, _ = httpGet(t, ordersURL+"/health/ready")
		if code == 503 {
			detected = true
			t.Log("Orders detected Redis failure (503)")
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !detected {
		t.Fatal("orders did not detect Redis failure within 30s")
	}

	// Wait for Redis to come back (deployment will recreate the pod)
	t.Log("Waiting for Redis recovery...")
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "app=redis", "--timeout=60s")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Redis did not recover: %s\n%s", err, out)
	}

	// Wait for orders to recover
	t.Log("Waiting for orders to recover...")
	deadline = time.Now().Add(30 * time.Second)
	recovered := false
	for time.Now().Before(deadline) {
		code, _ = httpGet(t, ordersURL+"/health/ready")
		if code == 200 {
			recovered = true
			t.Log("Orders recovered (200)")
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !recovered {
		t.Fatal("orders did not recover within 30s after Redis came back")
	}
}

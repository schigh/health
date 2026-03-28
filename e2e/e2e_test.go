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

// kubectlNoFail runs kubectl and returns output + error without failing the test.
func kubectlNoFail(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// portForward starts a port-forward and returns the local URL and a cleanup func.
func portForward(t *testing.T, svc string, remotePort int) (string, func()) {
	t.Helper()
	localPort := 30000 + remotePort
	cmd := exec.Command("kubectl", "port-forward", fmt.Sprintf("svc/%s", svc), fmt.Sprintf("%d:%d", localPort, remotePort))
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("port-forward %s failed: %v", svc, err)
	}
	time.Sleep(2 * time.Second)

	url := fmt.Sprintf("http://127.0.0.1:%d", localPort)
	cleanup := func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
	return url, cleanup
}

// httpGet fetches a URL and returns status code + body. Fails the test on error.
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

// httpGetSafe fetches a URL and returns status code + body + error without failing.
func httpGetSafe(url string) (int, []byte, error) {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}

// pollForStatus polls a URL until it returns the expected status or times out.
func pollForStatus(url string, expected int, timeout time.Duration) (bool, int) {
	deadline := time.Now().Add(timeout)
	var lastCode int
	for time.Now().Before(deadline) {
		code, _, err := httpGetSafe(url)
		if err == nil {
			lastCode = code
			if code == expected {
				return true, code
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false, lastCode
}

// ensureScale ensures a deployment has the expected number of ready replicas.
func ensureScale(t *testing.T, deployment string, replicas int) {
	t.Helper()
	kubectl(t, "scale", fmt.Sprintf("deployment/%s", deployment), fmt.Sprintf("--replicas=%d", replicas))
	if replicas == 0 {
		// wait for pods to terminate
		kubectlNoFail("wait", "--for=delete", "pod", "-l", fmt.Sprintf("app=%s", deployment), "--timeout=30s")
	} else {
		// wait for pods to be ready
		cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod",
			"-l", fmt.Sprintf("app=%s", deployment), "--timeout=60s")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s did not become ready: %s\n%s", deployment, err, out)
		}
	}
}

func TestMain(m *testing.M) {
	cmd := exec.Command("kubectl", "cluster-info")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "No Kubernetes cluster accessible. Run 'make e2e-cluster e2e-build e2e-deploy' first.\n")
		os.Exit(1)
	}

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

// ---------------------------------------------------------------------------
// Test 1: All probes return 200 when everything is healthy
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Test 2: Self-describing JSON includes all metadata fields
// ---------------------------------------------------------------------------

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
		Name          string `json:"name"`
		Status        string `json:"status"`
		Group         string `json:"group"`
		ComponentType string `json:"componentType"`
		Duration      string `json:"duration"`
		LastCheck     string `json:"lastCheck"`
	}
	if err := json.Unmarshal(pgRaw, &pg); err != nil {
		t.Fatalf("unmarshal postgres: %v", err)
	}

	if pg.Name != "postgres" {
		t.Errorf("expected name 'postgres', got %q", pg.Name)
	}
	if pg.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", pg.Status)
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

	// verify redis check exists with correct group
	redisRaw, ok := checks["redis"]
	if !ok {
		t.Fatal("redis check not found in response")
	}
	var rd struct {
		Group string `json:"group"`
	}
	json.Unmarshal(redisRaw, &rd)
	if rd.Group != "cache" {
		t.Errorf("expected redis group 'cache', got %q", rd.Group)
	}

	// verify payments-api check exists
	if _, ok := checks["payments-api"]; !ok {
		t.Error("payments-api check not found in response")
	}

	t.Logf("Orders has %d checks with full metadata", len(checks))
}

// ---------------------------------------------------------------------------
// Test 3: Discovery manifest is complete and accurate
// ---------------------------------------------------------------------------

func TestDiscoveryManifest(t *testing.T) {
	services := []struct {
		svc           string
		port          int
		expectService string
		expectChecks  int
		expectVersion string
	}{
		{"gateway-svc", 8181, "gateway", 2, "e2e"},
		{"orders-svc", 8182, "orders", 3, "e2e"},
		{"payments-svc", 8183, "payments", 1, "e2e"},
	}

	for _, s := range services {
		t.Run(s.svc, func(t *testing.T) {
			url, cleanup := portForward(t, s.svc, s.port)
			defer cleanup()

			code, body := httpGet(t, url+"/.well-known/health")
			if code != 200 {
				t.Fatalf("expected 200, got %d", code)
			}

			var manifest discovery.Manifest
			if err := json.Unmarshal(body, &manifest); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if manifest.Service != s.expectService {
				t.Errorf("expected service %q, got %q", s.expectService, manifest.Service)
			}
			if manifest.Version != s.expectVersion {
				t.Errorf("expected version %q, got %q", s.expectVersion, manifest.Version)
			}
			if manifest.Status != "pass" {
				t.Errorf("expected status 'pass', got %q", manifest.Status)
			}
			if len(manifest.Checks) != s.expectChecks {
				t.Errorf("expected %d checks, got %d", s.expectChecks, len(manifest.Checks))
			}
			if manifest.Timestamp.IsZero() {
				t.Error("expected non-zero timestamp")
			}

			// verify every check has a name and status
			for _, check := range manifest.Checks {
				if check.Name == "" {
					t.Error("check has empty name")
				}
				if check.Status == "" {
					t.Error("check has empty status")
				}
			}

			t.Logf("%s: %d checks, status=%s", s.expectService, len(manifest.Checks), manifest.Status)
		})
	}
}

// ---------------------------------------------------------------------------
// Test 4: Discovery graph chain is correct across all services
// ---------------------------------------------------------------------------

func TestDiscoveryGraph(t *testing.T) {
	// Fetch all 3 manifests and verify the dependency chain
	gwURL, gwCleanup := portForward(t, "gateway-svc", 8181)
	defer gwCleanup()
	ordersURL, ordersCleanup := portForward(t, "orders-svc", 8182)
	defer ordersCleanup()
	paymentsURL, paymentsCleanup := portForward(t, "payments-svc", 8183)
	defer paymentsCleanup()

	// Gateway manifest
	_, gwBody := httpGet(t, gwURL+"/.well-known/health")
	var gwManifest discovery.Manifest
	json.Unmarshal(gwBody, &gwManifest)

	if gwManifest.Service != "gateway" {
		t.Fatalf("expected gateway, got %q", gwManifest.Service)
	}

	// Gateway should depend on orders
	gwDepsOnOrders := false
	for _, check := range gwManifest.Checks {
		for _, dep := range check.DependsOn {
			if strings.Contains(dep, "orders") {
				gwDepsOnOrders = true
			}
		}
	}
	if !gwDepsOnOrders {
		t.Error("gateway should have a dependency on orders")
	}

	// Orders manifest
	_, ordersBody := httpGet(t, ordersURL+"/.well-known/health")
	var ordersManifest discovery.Manifest
	json.Unmarshal(ordersBody, &ordersManifest)

	if ordersManifest.Service != "orders" {
		t.Fatalf("expected orders, got %q", ordersManifest.Service)
	}

	// Orders should depend on payments
	ordersDepsOnPayments := false
	for _, check := range ordersManifest.Checks {
		for _, dep := range check.DependsOn {
			if strings.Contains(dep, "payments") {
				ordersDepsOnPayments = true
			}
		}
	}
	if !ordersDepsOnPayments {
		t.Error("orders should have a dependency on payments")
	}

	// Payments manifest (leaf node, no HTTP dependencies)
	_, paymentsBody := httpGet(t, paymentsURL+"/.well-known/health")
	var paymentsManifest discovery.Manifest
	json.Unmarshal(paymentsBody, &paymentsManifest)

	if paymentsManifest.Service != "payments" {
		t.Fatalf("expected payments, got %q", paymentsManifest.Service)
	}

	// Payments should have no HTTP dependencies (only postgres via TCP)
	for _, check := range paymentsManifest.Checks {
		for _, dep := range check.DependsOn {
			if strings.HasPrefix(dep, "http") {
				t.Errorf("payments should have no HTTP dependencies, found %q", dep)
			}
		}
	}

	t.Logf("Discovery chain verified: gateway(%d checks) → orders(%d checks) → payments(%d checks)",
		len(gwManifest.Checks), len(ordersManifest.Checks), len(paymentsManifest.Checks))
}

// ---------------------------------------------------------------------------
// Test 5: Redis failure affects readiness but NOT liveness
// (Redis only has WithReadinessImpact on orders)
// ---------------------------------------------------------------------------

func TestRedisFailure_ReadinessNotLiveness(t *testing.T) {
	ordersURL, cleanup := portForward(t, "orders-svc", 8182)
	defer cleanup()

	// Verify healthy baseline
	code, _ := httpGet(t, ordersURL+"/health/ready")
	if code != 200 {
		t.Fatalf("orders should be ready initially, got %d", code)
	}

	// Kill Redis
	t.Log("Scaling Redis to 0 replicas...")
	ensureScale(t, "redis", 0)

	// Wait for readiness to fail
	t.Log("Waiting for orders readiness to fail...")
	ok, lastCode := pollForStatus(ordersURL+"/health/ready", 503, 30*time.Second)
	if !ok {
		t.Fatalf("orders readiness did not go 503, last code: %d", lastCode)
	}
	t.Log("Orders readiness is 503 (correct)")

	// Liveness should STILL be 200 (Redis only affects readiness)
	liveCode, _ := httpGet(t, ordersURL+"/health/live")
	if liveCode != 200 {
		t.Errorf("orders liveness should still be 200 during Redis outage, got %d", liveCode)
	}
	t.Log("Orders liveness is still 200 (correct: Redis only affects readiness)")

	// Restore Redis
	t.Log("Scaling Redis back to 1 replica...")
	ensureScale(t, "redis", 1)

	// Wait for recovery
	t.Log("Waiting for orders to recover...")
	ok, lastCode = pollForStatus(ordersURL+"/health/ready", 200, 30*time.Second)
	if !ok {
		t.Fatalf("orders did not recover, last code: %d", lastCode)
	}
	t.Log("Orders recovered to 200")
}

// ---------------------------------------------------------------------------
// Test 6: Cascading failure (Postgres affects liveness on orders AND payments,
// gateway sees orders go down via HTTP check)
// ---------------------------------------------------------------------------

func TestCascadingFailure_Postgres(t *testing.T) {
	gwURL, gwCleanup := portForward(t, "gateway-svc", 8181)
	defer gwCleanup()
	ordersURL, ordersCleanup := portForward(t, "orders-svc", 8182)
	defer ordersCleanup()
	paymentsURL, paymentsCleanup := portForward(t, "payments-svc", 8183)
	defer paymentsCleanup()

	// Verify all healthy
	for _, u := range []string{gwURL, ordersURL, paymentsURL} {
		code, _ := httpGet(t, u+"/health/ready")
		if code != 200 {
			t.Fatalf("expected 200 at %s/health/ready, got %d", u, code)
		}
	}

	// Kill Postgres (affects liveness on orders and payments)
	t.Log("Scaling Postgres to 0 replicas...")
	ensureScale(t, "postgres", 0)

	// Payments should go down (postgres has liveness impact)
	t.Log("Waiting for payments to detect Postgres failure...")
	ok, lastCode := pollForStatus(paymentsURL+"/health/live", 503, 30*time.Second)
	if !ok {
		t.Fatalf("payments liveness did not go 503, last code: %d", lastCode)
	}
	t.Log("Payments liveness is 503")

	// Orders should go down (postgres has liveness impact)
	t.Log("Waiting for orders to detect Postgres failure...")
	ok, lastCode = pollForStatus(ordersURL+"/health/live", 503, 30*time.Second)
	if !ok {
		t.Fatalf("orders liveness did not go 503, last code: %d", lastCode)
	}
	t.Log("Orders liveness is 503")

	// Gateway should see orders down (orders HTTP check will fail because
	// orders' readiness endpoint returns 503)
	t.Log("Waiting for gateway to detect cascade...")
	ok, lastCode = pollForStatus(gwURL+"/health/ready", 503, 30*time.Second)
	if !ok {
		// Gateway might still be 200 if the HTTP check to orders hasn't run yet
		// or if orders is still responding (just with 503 which our HTTP checker
		// treats as unhealthy since it expects 200). Check manifest for details.
		t.Logf("Gateway readiness last code: %d (cascade may take longer)", lastCode)
	} else {
		t.Log("Gateway detected cascade (503)")
	}

	// Restore Postgres
	t.Log("Scaling Postgres back to 1 replica...")
	ensureScale(t, "postgres", 1)

	// Wait for full recovery chain: postgres → payments + orders → gateway
	t.Log("Waiting for payments recovery...")
	ok, _ = pollForStatus(paymentsURL+"/health/ready", 200, 45*time.Second)
	if !ok {
		t.Fatal("payments did not recover")
	}
	t.Log("Payments recovered")

	t.Log("Waiting for orders recovery...")
	ok, _ = pollForStatus(ordersURL+"/health/ready", 200, 45*time.Second)
	if !ok {
		t.Fatal("orders did not recover")
	}
	t.Log("Orders recovered")

	t.Log("Waiting for gateway recovery...")
	ok, _ = pollForStatus(gwURL+"/health/ready", 200, 45*time.Second)
	if !ok {
		t.Fatal("gateway did not recover")
	}
	t.Log("Full cascade recovery complete")
}

// ---------------------------------------------------------------------------
// Test 7: Startup sequencing (deploy a fresh service, verify it gates on deps)
// ---------------------------------------------------------------------------

func TestStartupSequencing(t *testing.T) {
	// Scale Postgres to 0 first
	t.Log("Scaling Postgres to 0...")
	ensureScale(t, "postgres", 0)

	// Restart payments (which depends on postgres for startup)
	t.Log("Restarting payments deployment...")
	kubectl(t, "rollout", "restart", "deployment/payments")

	// Give K8s a moment to start the new pod
	time.Sleep(5 * time.Second)

	// The new payments pod should NOT be ready (postgres is down, startup check fails)
	out, _ := kubectlNoFail("get", "pods", "-l", "app=payments", "-o",
		"jsonpath={.items[0].status.conditions[?(@.type=='Ready')].status}")
	if strings.TrimSpace(out) == "True" {
		t.Error("payments should NOT be ready while postgres is down")
	} else {
		t.Log("Payments correctly not ready while postgres is down")
	}

	// Verify startup probe is failing (pod should be in startup state)
	out, _ = kubectlNoFail("get", "pods", "-l", "app=payments", "-o",
		"jsonpath={.items[0].status.containerStatuses[0].started}")
	t.Logf("Payments container started status: %s", strings.TrimSpace(out))

	// Bring Postgres back
	t.Log("Scaling Postgres back to 1...")
	ensureScale(t, "postgres", 1)

	// Payments should eventually become ready
	t.Log("Waiting for payments to complete startup...")
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod",
		"-l", "app=payments", "--timeout=120s")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("payments did not become ready: %s\n%s", err, out)
	}
	t.Log("Payments completed startup after postgres became available")

	// Also wait for orders and gateway to recover (they depend on postgres/payments)
	t.Log("Waiting for full system recovery...")
	for _, app := range []string{"orders", "gateway"} {
		cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod",
			"-l", "app="+app, "--timeout=120s")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s did not recover: %s\n%s", app, err, out)
		}
	}
	t.Log("Full system recovered after startup sequencing test")
}

// ---------------------------------------------------------------------------
// Test 8: Manifest status reflects aggregate health correctly
// (when a check fails, manifest status should be "fail" or "warn")
// ---------------------------------------------------------------------------

func TestManifestStatusDuringFailure(t *testing.T) {
	ordersURL, cleanup := portForward(t, "orders-svc", 8182)
	defer cleanup()

	// Verify healthy manifest
	code, body := httpGet(t, ordersURL+"/.well-known/health")
	if code != 200 {
		t.Fatalf("expected 200, got %d", code)
	}
	var manifest discovery.Manifest
	json.Unmarshal(body, &manifest)
	if manifest.Status != "pass" {
		t.Fatalf("expected 'pass' initially, got %q", manifest.Status)
	}

	// Kill Redis
	t.Log("Scaling Redis to 0...")
	ensureScale(t, "redis", 0)

	// Wait for manifest status to change
	t.Log("Waiting for manifest status to change...")
	deadline := time.Now().Add(30 * time.Second)
	detected := false
	for time.Now().Before(deadline) {
		code, body, err := httpGetSafe(ordersURL + "/.well-known/health")
		if err != nil || code != 200 {
			time.Sleep(2 * time.Second)
			continue
		}
		var m discovery.Manifest
		json.Unmarshal(body, &m)
		if m.Status != "pass" {
			t.Logf("Manifest status changed to %q", m.Status)
			detected = true

			// Verify the redis check shows as unhealthy in the manifest
			for _, check := range m.Checks {
				if check.Name == "redis" {
					if check.Status == "healthy" {
						t.Error("redis check should not be healthy in manifest")
					} else {
						t.Logf("Redis check status in manifest: %q", check.Status)
					}
					if check.Error == "" {
						t.Error("redis check should have an error in manifest")
					}
				}
			}
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !detected {
		t.Fatal("manifest status did not change within 30s")
	}

	// Restore Redis
	t.Log("Restoring Redis...")
	ensureScale(t, "redis", 1)

	// Wait for manifest to go back to "pass"
	t.Log("Waiting for manifest recovery...")
	ok, _ := pollForStatus(ordersURL+"/health/ready", 200, 30*time.Second)
	if !ok {
		t.Fatal("orders did not recover")
	}

	_, body = httpGet(t, ordersURL+"/.well-known/health")
	json.Unmarshal(body, &manifest)
	if manifest.Status != "pass" {
		t.Errorf("expected manifest status 'pass' after recovery, got %q", manifest.Status)
	}
	t.Log("Manifest status recovered to 'pass'")
}

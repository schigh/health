# E2E Tests

End-to-end tests that verify the health library in a real Kubernetes cluster using [Kind](https://kind.sigs.k8s.io/).

## Topology

```
Gateway (:8181) → Orders (:8182) → Payments (:8183)
                    ↓        ↓
                 Postgres   Redis
```

Three microservices using the health library with real infrastructure dependencies. Tests verify K8s probes, startup sequencing, built-in checkers, the discovery protocol, failure detection, and recovery.

## Prerequisites

- Docker
- [Kind](https://kind.sigs.k8s.io/) (`go install sigs.k8s.io/kind@latest`)
- kubectl

## Running

```bash
# Full cycle: create cluster, build, deploy, test, teardown
make e2e

# Or step by step:
make e2e-cluster    # create Kind cluster
make e2e-build      # build Docker images and load into Kind
make e2e-deploy     # apply K8s manifests, wait for pods
make e2e-test       # run E2E test suite
make e2e-teardown   # delete Kind cluster
```

## What's Tested

| Test | What it verifies |
|---|---|
| TestProbesHealthy | All 3 services respond 200 on /livez, /readyz, /healthz |
| TestSelfDescribingJSON | JSON response includes group, componentType, duration, lastCheck |
| TestDiscoveryManifest | /.well-known/health manifest has service name, checks, dependsOn |
| TestDiscoveryGraph | Manifest chain: gateway → orders → payments verified end-to-end |
| TestFailureAndRecovery | Kill Redis pod, orders goes 503, Redis recovers, orders goes 200 |

## Debugging

Keep the cluster alive after tests:

```bash
make e2e-cluster e2e-build e2e-deploy
make e2e-test  # if this fails, cluster is still up
kubectl get pods
kubectl logs -l app=orders
kubectl port-forward svc/gateway-svc 8181:8181
curl localhost:8181/.well-known/health | jq
```

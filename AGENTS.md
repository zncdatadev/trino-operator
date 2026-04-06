<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-04-06 -->

# trino-operator

## Purpose
Manages Trino deployments on Kubernetes. Handles creation, configuration, and lifecycle management of Trino clusters for distributed SQL query execution. Supports coordinator and worker roles, TLS, authentication, and catalog management.

## Key Files
| File | Description |
|------|-------------|
| `go.mod` | Go module dependencies |
| `Makefile` | Build and development commands |
| `PROJECT` | Kubebuilder project metadata |
| `Dockerfile` | Operator container image build |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `api/v1alpha1/` | Kubernetes CRD definitions (`TrinoCluster`, `TrinoCatalog`) |
| `cmd/` | Operator entry point |
| `config/` | Kubernetes manifests and kustomize configs |
| `internal/controller/` | Controller and reconciliation logic |
| `internal/controller/cluster/` | Cluster-level reconciler |
| `internal/controller/coordinator/` | Coordinator role reconciler |
| `internal/controller/worker/` | Worker role reconciler |
| `internal/controller/common/` | Shared resources (configmap, statefulset, service, secret, authz) |
| `deploy/` | Deployment manifests |
| `examples/` | Example CR manifests |
| `test/` | E2E test suites |

## For AI Agents

### Working In This Directory
- Standard Kubebuilder operator structure
- Uses `operator-go` framework (`github.com/zncdatadev/operator-go`) for reconciliation
- Run `make test` for unit tests
- Run `make deploy` to deploy to cluster
- CRD group: `trino.kubedoop.dev`
- Two CRDs: `TrinoCluster` (main cluster) and `TrinoCatalog` (catalog connectors)

### Testing Requirements
- Unit/controller tests: `make test`
- E2E tests in `test/e2e/` — requires a running Kubernetes cluster
- Uses Ginkgo/Gomega test framework

### Common Patterns
- Main reconciler: `internal/controller/trino_controller.go` (`TrinoReconciler`)
- Cluster reconciler: `internal/controller/cluster/cluster.go`
- Role reconcilers: `coordinator/role.go`, `worker/role.go`
- Shared resources under `internal/controller/common/`
- CRDs follow `v1alpha1` API version
- Follows `operator-go` `GenericReconciler` pattern
- RBAC annotations defined in `trino_controller.go`

## Dependencies

### Internal
- `../operator-go` — Shared operator framework (`github.com/zncdatadev/operator-go v0.12.x`)

### External
- `sigs.k8s.io/controller-runtime` v0.23+
- `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go` v0.35+
- Go 1.25+
- Kubernetes 1.26+

<!-- MANUAL: -->

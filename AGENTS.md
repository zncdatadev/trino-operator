<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-04-06 | Updated: 2026-09-15 -->

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
| `internal/controller/` | GenericReconciler product integration, role-group resource handler, cluster extension, and configuration defaults |
| `internal/controller/common/authz/` | AuthenticationClass validation and Trino authentication configuration |
| `deploy/` | Deployment manifests |
| `docs/` | Operator behavior and upgrade procedures |
| `examples/` | Example CR manifests |
| `hack/` | Generated-manifest checks and framework upgrade acceptance scripts |
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
- Controller registration: `cmd/main.go` (`GenericReconciler`)
- Role and role-group reconciliation: `internal/controller/trino_handler.go` (`TrinoRoleGroupHandler`)
- Cluster-scoped shared resources: `internal/controller/cluster_extension.go` (`ClusterResourcesExtension`)
- Product configuration defaults and merge schemas: `internal/controller/config.go`
- Authentication handling: `internal/controller/common/authz/`
- CRDs follow `v1alpha1` API version
- Generated CRDs must stay synchronized with `deploy/helm/trino-operator/crds/crds.yaml`
- RBAC source is `config/rbac/role.yaml`; keep the Helm ClusterRole synchronized with `make helm-rbac-sync-check`

## Dependencies

### Internal
- `../operator-go` — Shared operator framework (`github.com/zncdatadev/operator-go v0.13.x`)

### External
- `sigs.k8s.io/controller-runtime` v0.23+
- `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go` v0.35+
- Go 1.25+
- Kubernetes 1.33-1.35 in release E2E; local and pull-request E2E default to 1.35

### AI Worktree Development Mode

**IMPORTANT**: When making code changes, work in a worktree under `.worktree/`, NOT in the main working directory.

#### Workflow
1. Create worktree: `git worktree add .worktree/<branch-name> -b <branch-name>`
2. Work in `.worktree/<branch-name>/` directory
3. Test: `cd .worktree/<branch-name> && make lint && make test`
4. Commit changes in the worktree
5. Push and create PR from the worktree branch
6. Cleanup: `git worktree remove .worktree/<branch-name>`

#### Rules
- NEVER modify files directly in the main working directory
- Each task gets its own worktree with a descriptive branch name
- Run `make generate` if API structs are modified
- Run `make lint && make test` before committing

<!-- MANUAL: -->

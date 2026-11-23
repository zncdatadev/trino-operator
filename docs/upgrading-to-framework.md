# Upgrading to the operator-go v0.13.0 framework

This migration requires a maintenance window. The framework keeps the existing
StatefulSet names, but changes their immutable selectors and governing Service
names. Kubernetes therefore cannot apply the new StatefulSets over workloads
created by the fixed pre-framework baseline. Stop Trino traffic and the old
operator, retain the data PVCs, and recreate the coordinator and worker
StatefulSets with the new operator.

Keep the TrinoCluster name, namespace, role-group names, Trino product version,
and storage settings unchanged during this procedure. Schedule product upgrades
and role-group changes separately.

## Intentional resource and API changes

| Contract | Before | Framework |
| ------- | ------ | --------- |
| StatefulSet name | `<cluster>-<role>-<group>` | Unchanged |
| StatefulSet selector | Descriptive `app.kubernetes.io/*` labels | `trino.kubedoop.dev/{cluster,role,role-group}` identity labels |
| StatefulSet serviceName | `<cluster>-<role>-<group>` | `<cluster>-<role>-<group>-headless` |
| Pod management policy | `OrderedReady` | `Parallel` |
| `data` volumeClaimTemplate | One claim per pod | Name and requested storage retained |
| Role-group Service | `<cluster>-<role>-<group>` ClusterIP | Name and identity retained; selector updated in place |
| Metrics Service | `<cluster>-<role>-<group>-metrics` headless Service | Name and identity retained |
| Coordinator Service | `<cluster>-coordinator` | Name and identity retained |
| Headless governing Service | None | New `<cluster>-<role>-<group>-headless` Service |
| Workload ServiceAccount | Namespace `default` | `trinocluster-<cluster>` |
| PDB | One `<cluster>-<role>` PDB when enabled in `roleConfig` | Name retained; selector updated in place |
| Role-group ConfigMap | `<cluster>-<role>-<group>` | Name and identity retained; generated contents change with the framework pipeline |
| Managed-by label | `trino.kubedoop.dev` | `operator-go` |
| Pod construction | Product-owned builders | Framework security defaults, config merge, overrides, logging, and lifecycle wiring |
| TrinoCluster status | Legacy URL/name/type/generation fields | Legacy fields retained alongside generic conditions, observed generation, and role-group status |
| Image specification | Product-local public type | Public shape retained and adapted internally to the shared operator-go image specification |

The public `spec.coordinators` and `spec.workers` fields remain in place. The
generated CRD still changes because the shared image, override, and status
schemas evolve. Review the rendered CRD diff before upgrading. Explicitly pin
storage capacity, resources, logging, overrides, and the full product image when
you need to reproduce an existing cluster's effective configuration.

The framework merge pipeline now applies role and role-group
`configOverrides` to `config.properties`, `node.properties`,
`secret.properties`, `log.properties`, catalog properties, and
`jvm.config`. JVM override keys are complete option names: empty values emit
flag options, `-Xmx`/`-Xms`/`-Xss` values are concatenated, keys ending
in `:` or `=` are concatenated, and other values are emitted with `=`.
For example, `{"-Xmx": "4G", "-Dfeature.enabled": "true"}` renders
`-Xmx4G` and `-Dfeature.enabled=true`.

OIDC is the supported Trino authentication provider in this version. Static
password and LDAP AuthenticationClasses are rejected during reconciliation
because the operator does not yet generate Trino password authenticator files;
this prevents an authentication declaration from silently starting an
unauthenticated or invalid coordinator. OIDC also requires TLS on the Trino
cluster. HTTP identity providers and HTTPS providers using public Web PKI are
supported; insecure TLS verification and custom OIDC CA SecretClasses are
rejected until the operator can add those CAs to the Trino client truststore.

The legacy controller used `serverSecretClass` for both the public HTTPS and
internal communication certificates, even though the CRD also persisted a
defaulted `internalSecretClass: tls`. The framework keeps that behavior during
this migration and continues to use `serverSecretClass` for both certificates,
regardless of the stored `internalSecretClass` value. Activating separate
internal certificates requires a future compatibility mechanism that can
distinguish an explicit opt-in from already persisted, previously ignored data.

`catalogLabelSelector` and the standalone `TrinoCatalog` resource remain
under construction, as they were before this migration. Use
`clusterConfig.catalogProperties` for catalogs that the operator must render.

Trino copies generated files into its writable configuration directory during
startup. To roll pods when a generated ConfigMap changes, opt the cluster into
the commons-operator restarter:

```yaml
metadata:
  labels:
    restarter.kubedoop.dev/enable: "true"
```

For an `external-stable` listener, separately provision a server certificate
whose SAN covers the LoadBalancer hostname used by clients. The automatic
SecretClass scopes cover pod DNS, role-group Services, the stable coordinator
Service, and node addresses, but cannot predict a cloud-assigned hostname.

## Before the maintenance window

1. Save the TrinoCluster, generated resources, old operator image, deployment,
   RBAC, and CRD. Record every data PVC UID, bound PV name, and PV UID. Take any
   application-level backups required by the catalogs Trino accesses.
2. Run a representative query and confirm that the expected workers are active.
3. Review the StatefulSet, ConfigMap, Service, PDB, ServiceAccount, and PVC diff
   for the real cluster. Include custom pod overrides and security contexts.
4. Rehearse the procedure on a representative copy. The repository acceptance
   test uses one coordinator, one worker, default kind storage, and `SELECT 1`.

## Upgrade procedure

Use an explicit kubeconfig and namespace in every command. Substitute the
operator namespace and deployment name if Helm installed them elsewhere.

```bash
export KUBECONFIG=/path/to/cluster.kubeconfig
NS=my-trino-namespace
CLUSTER=my-trino
OPERATOR_NS=trino-operator-system
OPERATOR_DEPLOYMENT=trino-operator-controller-manager

kubectl -n "$OPERATOR_NS" scale deployment "$OPERATOR_DEPLOYMENT" --replicas=0
kubectl -n "$OPERATOR_NS" wait --for=delete pod \
  -l control-plane=controller-manager --timeout=180s
```

For every coordinator and worker role group, set the StatefulSet PVC retention
policy to `Retain`, scale it to zero, wait for all pods to terminate, and delete
the StatefulSet. Keep its existing Services, ConfigMap, and data PVCs so the
framework can update the mutable resources in place, add the governing headless
Service, and reattach the same volumes.

```bash
ROLE=worker
GROUP=default
STS="$CLUSTER-$ROLE-$GROUP"
REPLICAS=$(kubectl -n "$NS" get sts "$STS" -o jsonpath='{.spec.replicas}')
kubectl -n "$NS" patch sts "$STS" --type merge \
  -p '{"spec":{"persistentVolumeClaimRetentionPolicy":{"whenDeleted":"Retain","whenScaled":"Retain"}}}'
kubectl -n "$NS" scale sts "$STS" --replicas=0
for ((ordinal=0; ordinal<REPLICAS; ordinal++)); do
  kubectl -n "$NS" wait --for=delete "pod/$STS-$ordinal" --timeout=180s
done
kubectl -n "$NS" delete sts "$STS" --wait=true
```

Install the new CRD, RBAC, and operator. The framework recreates the StatefulSets
with the same names and `data` claim templates, so Kubernetes reattaches the
existing claims. Wait for the coordinator and workers, then compare PVC UIDs,
bound PV names, and PV UIDs with the saved record. Run the pre-upgrade query and
a new query before restoring traffic. Remove no PVC during this process.

The framework also creates `trinocluster-<cluster>` and uses it in both pod
templates. The namespace `default` ServiceAccount can remain for other workloads.

## Stopping and resuming

After the upgrade, verify the operational stop contract before relying on it:

```bash
kubectl -n "$NS" patch trinocluster "$CLUSTER" --type merge \
  -p '{"spec":{"clusterOperation":{"stopped":true}}}'
# Wait until every generated StatefulSet has spec.replicas=0 and no role pods remain.

kubectl -n "$NS" patch trinocluster "$CLUSTER" --type merge \
  -p '{"spec":{"clusterOperation":{"stopped":false}}}'
# Wait for all replicas, compare resource and PVC identities, and run a query again.
```

Stopping scales the StatefulSets. It should preserve the StatefulSet, ConfigMap,
Service, PDB, ServiceAccount, PVC, and PV identities. A configuration or product
version change while stopped can still cause a rollout when the cluster resumes.

## Rollback

Rollback repeats the outage and StatefulSet recreation. Stop the framework
operator, retain the PVCs, scale both roles to zero, wait for all pods to
terminate, and delete their StatefulSets and new `-headless` Services. Delete the
unused `trinocluster-<cluster>` ServiceAccount after no pod references it. Keep
the role PDBs so the old controller can update their selectors in place. Redeploy
the saved old controller and RBAC while retaining the expanded CRD. Replacing the
CRD with the old schema can prune persisted fields.

The old operator recreates its workload shape around the retained data PVCs.
Confirm that the PVC UIDs and PV bindings still match, both roles become ready,
and a Trino query succeeds before restoring traffic. A controller rollback does
not reverse a simultaneous Trino product or catalog data-format upgrade.

## Executable acceptance and evidence

The acceptance target requires the Go version from `go.mod`, Docker, kind,
kubectl, Helm, and Python 3. It creates a dedicated kind cluster and refuses to
reuse an existing cluster or a cluster containing TrinoClusters. The old source
is fixed at commit `6aee790ccc7daab3a13a6b45f87fffe1fc937a98`, which uses
operator-go v0.12.6.

```bash
GOTOOLCHAIN=go1.25.8 make framework-upgrade-e2e \
  CHAINSAW_CLUSTER=trino-framework-upgrade \
  CHAINSAW_KUBECONFIG=.kubeconfig-upgrade \
  KIND_K8S_VERSION=1.35.0 PRODUCT_VERSION=451
```

The script creates one coordinator and one worker with explicit 1 GiB data
claims and role-level PDB configuration. It runs `SELECT 1` and verifies the
active worker count on the baseline, after upgrade, after a stopped/resumed
cycle, and after rollback. It captures `sts`, `cm`, `svc`, `pdb`, `sa`, and `pvc`
at every phase, verifies PVC/PV identity, and checks the expected selector,
governing Service, pod management, ServiceAccount, managed-by, and PDB
transitions. It also seeds the legacy URL/name/type/generation status fields and
verifies that the framework preserves them while adding current conditions,
observed generation, and role-group status. Raw resources, normalized diffs,
query results, operator manifests, events, and `result.txt` are stored under
`upgrade-evidence/`. CI uploads that directory even when the test fails.

The fixture exercises default kind storage and unauthenticated HTTP. Validate
TLS, authentication, external listeners, Vector, custom storage classes, catalog
connectivity, and site-specific pod overrides separately when a deployment uses
them. Vector uses Kubernetes native sidecars, so use Kubernetes 1.29 or newer;
the repository and CI default to 1.35.0.

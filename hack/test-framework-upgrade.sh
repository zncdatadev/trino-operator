#!/usr/bin/env bash
# Destructive acceptance test confined to an empty, explicitly selected kind cluster.
set -euo pipefail

: "${KUBECONFIG:?Set KUBECONFIG to the disposable kind cluster}"
: "${CHAINSAW_CLUSTER:?Set the disposable kind cluster name}"
: "${UPGRADE_BASELINE_DIR:?Provide a checkout of the pre-framework operator}"
: "${UPGRADE_BASELINE_IMAGE:?Build the pre-framework image first}"
: "${IMG:?Build the framework image first}"

root=$(cd "$(dirname "$0")/.." && pwd)
product_version=${PRODUCT_VERSION:-451}
query_image=${UPGRADE_QUERY_IMAGE:-quay.io/zncdatadev/testing-tools:0.1.0-kubedoop0.0.0-dev}
evidence=${UPGRADE_EVIDENCE_DIR:-$root/upgrade-evidence}
mkdir -p "$evidence"
evidence=$(cd "$evidence" && pwd)
evidence=$(mktemp -d "$evidence/run.XXXXXX")
echo "Migration evidence: $evidence"

test "$(kubectl config current-context)" = "kind-$CHAINSAW_CLUSTER"
crd=$(kubectl get crd trinoclusters.trino.kubedoop.dev --ignore-not-found -o name)
clusters=
if [ -n "$crd" ]; then
    clusters=$(kubectl get trinoclusters -A -o name)
fi
if [ -n "$clusters" ]; then
    echo 'Use an empty kind cluster: existing TrinoClusters would be affected.' >&2
    exit 1
fi

namespace=trino-framework-upgrade
if kubectl get namespace "$namespace" >/dev/null 2>&1; then
    echo "Namespace $namespace already exists; inspect the previous run before retrying." >&2
    exit 1
fi

scratch=$(mktemp -d)
k() { kubectl -n "$namespace" "$@"; }

finish() {
    local result=$?
    if (( result != 0 )); then
        k get trinocluster,sts,pod,pvc,cm,svc,pdb,sa -o yaml \
            > "$evidence/failure-resources.yaml" 2>&1 || true
        k get events --sort-by=.lastTimestamp \
            > "$evidence/failure-events.txt" 2>&1 || true
        kubectl -n trino-operator-system logs deployment/trino-operator-controller-manager \
            --tail=300 > "$evidence/failure-operator.log" 2>&1 || true
    fi
    rm -rf "$scratch"
}
trap finish EXIT

wait_for() {
    local description=$1
    shift
    local deadline=$((SECONDS + 600))
    until "$@"; do
        if (( SECONDS >= deadline )); then
            echo "Timed out: $description" >&2
            k get sts,pod,pvc,svc,pdb -o wide || true
            k get events --sort-by=.lastTimestamp || true
            return 1
        fi
        sleep 5
    done
}

ready() {
    k get sts "$1" -o json | python3 -c '
import json, sys

statefulset = json.load(sys.stdin)
status = statefulset.get("status", {})
assert statefulset["spec"]["replicas"] == 1
assert status.get("readyReplicas", 0) == 1
assert status.get("observedGeneration", 0) >= statefulset["metadata"]["generation"]
' 2>/dev/null
}

scaled_to_zero() {
    local workload=$1
    k get sts "$workload" -o json | python3 -c '
import json, sys

statefulset = json.load(sys.stdin)
status = statefulset.get("status", {})
assert statefulset["spec"]["replicas"] == 0
assert status.get("readyReplicas", 0) == 0
' 2>/dev/null && ! k get pod "$workload-0" >/dev/null 2>&1
}

render_operator() {
    local source=$1 image=$2 output=$3
    local render
    render=$(mktemp -d "$scratch/render.XXXXXX")
    cp -R "$source/config" "$render/config"
    (cd "$render/config/manager" && "$root/bin/kustomize" edit set image "controller=$image")
    "$root/bin/kustomize" build "$render/config/default" > "$output"
}

deploy_operator() {
    kubectl apply -f "$1"
    kubectl -n trino-operator-system rollout status \
        deployment/trino-operator-controller-manager --timeout=300s
}

stop_operator() {
    kubectl -n trino-operator-system scale \
        deployment/trino-operator-controller-manager --replicas=0
    kubectl -n trino-operator-system wait --for=delete pod \
        -l control-plane=controller-manager --timeout=180s
}

replace_workloads() {
    stop_operator
    local role workload
    for role in coordinator worker; do
        workload="upgrade-$role-default"
        # Retain the data claims even if a user supplied a different StatefulSet policy.
        k patch sts "$workload" --type merge \
            -p '{"spec":{"persistentVolumeClaimRetentionPolicy":{"whenDeleted":"Retain","whenScaled":"Retain"}}}'
        k scale sts "$workload" --replicas=0
        if k get pod "$workload-0" >/dev/null 2>&1; then
            k wait --for=delete "pod/$workload-0" --timeout=180s
        fi
        k delete sts "$workload" --wait=true
        k delete svc "$workload-headless" --ignore-not-found
    done
}

snapshot() {
    local phase=$1 pvc_file pv_file
    pvc_file="$evidence/$phase-data-pvcs.json"
    pv_file="$evidence/$phase-data-pvs.json"

    k get sts,cm,svc,pdb,sa,pvc -o json > "$evidence/$phase-resources.json"
    k get trinocluster upgrade -o json > "$evidence/$phase-cluster.json"
    k get pvc \
        data-upgrade-coordinator-default-0 \
        data-upgrade-worker-default-0 \
        -o json > "$pvc_file"

    mapfile -t pv_names < <(python3 -c '
import json, sys

for claim in json.load(sys.stdin)["items"]:
    print(claim["spec"]["volumeName"])
' < "$pvc_file")
    test "${#pv_names[@]}" -eq 2
    kubectl get pv "${pv_names[@]}" -o json > "$pv_file"

    python3 - "$pvc_file" "$pv_file" > "$evidence/$phase-pvc-identities.json" <<'PY'
import json
import pathlib
import sys

claims = json.loads(pathlib.Path(sys.argv[1]).read_text())["items"]
volumes = {
    item["metadata"]["name"]: item
    for item in json.loads(pathlib.Path(sys.argv[2]).read_text())["items"]
}
identities = []
for claim in claims:
    volume_name = claim["spec"]["volumeName"]
    volume = volumes[volume_name]
    identities.append(
        {
            "claim": claim["metadata"]["name"],
            "claimUID": claim["metadata"]["uid"],
            "volume": volume_name,
            "volumeUID": volume["metadata"]["uid"],
            "volumeClaimRefUID": volume["spec"]["claimRef"]["uid"],
        }
    )
print(json.dumps(sorted(identities, key=lambda item: item["claim"]), sort_keys=True))
PY
}

legacy_status_present() {
    k get trinocluster upgrade -o json | python3 -c '
import json
import sys

status = json.load(sys.stdin).get("status", {})
assert status.get("generation") == 4242
assert status.get("name") == "upgrade"
assert status.get("type") == "trino"
assert status.get("urls") == [
    {"name": "legacy-coordinator", "url": "http://upgrade-coordinator:8080"}
]
' 2>/dev/null
}

seed_legacy_status() {
    k patch trinocluster upgrade --subresource=status --type merge -p '{
      "status": {
        "generation": 4242,
        "name": "upgrade",
        "type": "trino",
        "urls": [
          {
            "name": "legacy-coordinator",
            "url": "http://upgrade-coordinator:8080"
          }
        ]
      }
    }'
    wait_for "legacy status persisted" legacy_status_present
}

query_once() {
    local pod=$1 output=$2
    k exec "$pod" -- /usr/local/bin/python -c '
import json
import trino

connection = trino.dbapi.connect(
    host="upgrade-coordinator",
    port=8080,
    user="framework-upgrade-check",
    http_scheme="http",
    session_properties={"query_max_execution_time": "60s"},
)
cursor = connection.cursor()
cursor.execute("SELECT 1")
select_one = int(cursor.fetchone()[0])
cursor.execute("SELECT COUNT(*) FROM system.runtime.nodes WHERE coordinator = false")
active_workers = int(cursor.fetchone()[0])
assert select_one == 1
assert active_workers == 1
print(json.dumps({"activeWorkers": active_workers, "selectOne": select_one}, sort_keys=True))
' > "$output" 2>&1
}

run_query() {
    local phase=$1 pod="trino-query-$1" output="$evidence/$1-query.txt"
    k delete pod "$pod" --ignore-not-found --wait=true >/dev/null
    k run "$pod" --image="$query_image" --restart=Never --command -- sleep 1200
    k wait --for=condition=Ready "pod/$pod" --timeout=300s
    wait_for "$phase Trino query and worker registration" query_once "$pod" "$output"
    cat "$output"
    k delete pod "$pod" --wait=true >/dev/null
}

render_operator "$UPGRADE_BASELINE_DIR" "$UPGRADE_BASELINE_IMAGE" \
    "$evidence/baseline-operator.yaml"
render_operator "$root" "$IMG" "$evidence/framework-operator.yaml"
kind load docker-image --name "$CHAINSAW_CLUSTER" \
    "$UPGRADE_BASELINE_IMAGE" "$IMG"

deploy_operator "$evidence/baseline-operator.yaml"
kubectl create namespace "$namespace"
cat > "$evidence/input-cluster.yaml" <<YAML
apiVersion: trino.kubedoop.dev/v1alpha1
kind: TrinoCluster
metadata:
  name: upgrade
spec:
  image:
    productVersion: "$product_version"
  coordinators:
    roleConfig:
      podDisruptionBudget:
        enabled: true
        maxUnavailable: 1
    roleGroups:
      default:
        replicas: 1
        config:
          resources:
            storage:
              capacity: 1Gi
  workers:
    roleConfig:
      podDisruptionBudget:
        enabled: true
        maxUnavailable: 1
    roleGroups:
      default:
        replicas: 1
        config:
          resources:
            storage:
              capacity: 1Gi
YAML
k apply -f "$evidence/input-cluster.yaml"
for role in coordinator worker; do
    wait_for "baseline $role ready" ready "upgrade-$role-default"
done
run_query before
seed_legacy_status
snapshot before

echo 'Upgrade with the existing coordinator and worker data PVCs'
replace_workloads
deploy_operator "$evidence/framework-operator.yaml"
for role in coordinator worker; do
    wait_for "framework $role ready" ready "upgrade-$role-default"
done
run_query after
snapshot after
cmp "$evidence/before-pvc-identities.json" "$evidence/after-pvc-identities.json"

echo 'Exercise framework stop and resume without recreating resources'
k patch trinocluster upgrade --type merge \
    -p '{"spec":{"clusterOperation":{"stopped":true}}}'
for role in coordinator worker; do
    wait_for "stopped $role" scaled_to_zero "upgrade-$role-default"
done
snapshot stopped

k patch trinocluster upgrade --type merge \
    -p '{"spec":{"clusterOperation":{"stopped":false}}}'
for role in coordinator worker; do
    wait_for "resumed $role ready" ready "upgrade-$role-default"
done
run_query resumed
snapshot resumed
cmp "$evidence/after-pvc-identities.json" "$evidence/stopped-pvc-identities.json"
cmp "$evidence/after-pvc-identities.json" "$evidence/resumed-pvc-identities.json"

echo 'Rollback around the retained PVCs while keeping the expanded CRD'
replace_workloads
k delete sa trinocluster-upgrade --ignore-not-found

# Applying an older CRD can prune fields already persisted by the framework.
# Reapply the saved baseline deployment and RBAC while retaining the current CRD.
python3 - "$evidence/baseline-operator.yaml" "$scratch/rollback.yaml" <<'PY'
import pathlib
import sys

documents = pathlib.Path(sys.argv[1]).read_text().split("\n---\n")
pathlib.Path(sys.argv[2]).write_text(
    "\n---\n".join(
        document
        for document in documents
        if "\nkind: CustomResourceDefinition\n" not in "\n" + document
    )
)
PY
deploy_operator "$scratch/rollback.yaml"
for role in coordinator worker; do
    wait_for "rollback $role ready" ready "upgrade-$role-default"
done
run_query rollback
snapshot rollback
cmp "$evidence/before-pvc-identities.json" "$evidence/rollback-pvc-identities.json"

python3 "$root/hack/compare-upgrade-resources.py" "$evidence"
kubectl delete namespace "$namespace" --wait=true --timeout=300s
deploy_operator "$evidence/framework-operator.yaml"
printf '%s\n' \
    "PASS: Trino $product_version baseline -> framework -> stopped/resumed -> baseline; query and PVC/PV identities preserved" \
    | tee "$evidence/result.txt"

"""Check Trino migration contracts and save normalized resource diffs."""

import copy
import difflib
import json
import pathlib
import sys


PHASES = ("before", "after", "stopped", "resumed", "rollback")
ROLES = ("coordinator", "worker")


def resources(root: pathlib.Path, phase: str) -> dict[tuple[str, str], dict]:
    items = json.loads((root / f"{phase}-resources.json").read_text())["items"]
    return {(item["kind"], item["metadata"]["name"]): item for item in items}


def require(items: dict[tuple[str, str], dict], kind: str, name: str) -> dict:
    key = (kind, name)
    assert key in items, f"missing {kind} {name}"
    return items[key]


def uid(items: dict[tuple[str, str], dict], kind: str, name: str) -> str:
    return require(items, kind, name)["metadata"]["uid"]


def pvc_identities(root: pathlib.Path, phase: str) -> list[dict]:
    return json.loads((root / f"{phase}-pvc-identities.json").read_text())


def query_result(root: pathlib.Path, phase: str) -> dict:
    lines = (root / f"{phase}-query.txt").read_text().splitlines()
    assert lines, f"empty query result for {phase}"
    return json.loads(lines[-1])


def cluster(root: pathlib.Path, phase: str) -> dict:
    return json.loads((root / f"{phase}-cluster.json").read_text())


def assert_legacy_status(value: dict, phase: str) -> None:
    status = value.get("status", {})
    assert status.get("generation") == 4242, (phase, "legacy generation")
    assert status.get("name") == "upgrade", (phase, "legacy name")
    assert status.get("type") == "trino", (phase, "legacy type")
    assert status.get("urls") == [
        {
            "name": "legacy-coordinator",
            "url": "http://upgrade-coordinator:8080",
        }
    ], (phase, "legacy URLs")


def assert_framework_status(value: dict, phase: str, available: bool) -> None:
    status = value["status"]
    generation = value["metadata"]["generation"]
    assert status.get("observedGeneration") == generation, (
        phase,
        "observed generation",
    )
    assert status.get("roleGroups") == {
        "coordinator": ["default"],
        "worker": ["default"],
    }, (phase, "role groups")

    conditions = {
        condition["type"]: condition for condition in status.get("conditions", [])
    }
    for condition_type in (
        "Available",
        "Progressing",
        "Paused",
        "Degraded",
        "ReconcileComplete",
    ):
        assert condition_type in conditions, (phase, condition_type)
        assert conditions[condition_type]["observedGeneration"] == generation, (
            phase,
            condition_type,
            "observed generation",
        )
    assert conditions["Available"]["status"] == ("True" if available else "False"), (
        phase,
        "Available condition",
    )
    assert conditions["ReconcileComplete"]["status"] == "True", (
        phase,
        "ReconcileComplete condition",
    )


def claim_specs(statefulset: dict) -> dict[str, dict]:
    result = {}
    for claim in statefulset["spec"]["volumeClaimTemplates"]:
        spec = copy.deepcopy(claim["spec"])
        # The old builder wrote the Kubernetes default explicitly; the new builder
        # lets API defaulting supply the same filesystem value.
        spec.setdefault("volumeMode", "Filesystem")
        result[claim["metadata"]["name"]] = spec
    return result


def normalize(value: dict) -> dict:
    value = copy.deepcopy(value)
    value.pop("status", None)
    metadata = value["metadata"]
    for key in (
        "uid",
        "resourceVersion",
        "generation",
        "creationTimestamp",
        "managedFields",
    ):
        metadata.pop(key, None)
    metadata.get("annotations", {}).pop(
        "kubectl.kubernetes.io/last-applied-configuration", None
    )
    for owner in metadata.get("ownerReferences", []):
        owner.pop("uid", None)
    if value["kind"] == "Service":
        for key in ("clusterIP", "clusterIPs", "healthCheckNodePort"):
            value["spec"].pop(key, None)
        for port in value["spec"].get("ports", []):
            port.pop("nodePort", None)
    return value


def main(root: pathlib.Path) -> None:
    phase_resources = {phase: resources(root, phase) for phase in PHASES}
    before = phase_resources["before"]
    after = phase_resources["after"]
    stopped = phase_resources["stopped"]
    resumed = phase_resources["resumed"]
    rollback = phase_resources["rollback"]

    phase_clusters = {phase: cluster(root, phase) for phase in PHASES}
    for phase, value in phase_clusters.items():
        assert_legacy_status(value, phase)
    assert "observedGeneration" not in phase_clusters["before"].get("status", {})
    assert_framework_status(phase_clusters["after"], "after", available=True)
    assert_framework_status(phase_clusters["stopped"], "stopped", available=False)
    assert_framework_status(phase_clusters["resumed"], "resumed", available=True)
    # The old controller leaves unknown expanded status fields intact on rollback.
    assert_framework_status(phase_clusters["rollback"], "rollback", available=True)

    service_names = {"upgrade-coordinator"}
    config_map_names = set()
    workload_names = set()
    pvc_names = set()
    pdb_names = set()
    for role in ROLES:
        workload = f"upgrade-{role}-default"
        workload_names.add(workload)
        config_map_names.add(workload)
        pvc_names.add(f"data-{workload}-0")
        pdb_names.add(f"upgrade-{role}")
        service_names.update({workload, f"{workload}-metrics"})

    for phase, items in phase_resources.items():
        for name in workload_names:
            require(items, "StatefulSet", name)
            require(items, "ConfigMap", name)
            require(items, "PersistentVolumeClaim", f"data-{name}-0")
        for name in service_names:
            require(items, "Service", name)

    for phase in ("before", "rollback"):
        for name in workload_names:
            assert (
                "Service",
                f"{name}-headless",
            ) not in phase_resources[phase], (
                f"unexpected headless service in {phase}: {name}-headless"
            )
    for phase in ("after", "stopped", "resumed"):
        for name in workload_names:
            headless = require(
                phase_resources[phase], "Service", f"{name}-headless"
            )
            assert headless["spec"]["clusterIP"] == "None"

    assert ("ServiceAccount", "trinocluster-upgrade") not in before
    assert ("ServiceAccount", "trinocluster-upgrade") in after
    assert ("ServiceAccount", "trinocluster-upgrade") in stopped
    assert ("ServiceAccount", "trinocluster-upgrade") in resumed
    assert ("ServiceAccount", "trinocluster-upgrade") not in rollback

    for name in pdb_names:
        for phase, items in phase_resources.items():
            require(items, "PodDisruptionBudget", name)

    # ConfigMaps and Services are updated in place across both controller switches.
    for kind, names in (
        ("ConfigMap", config_map_names),
        ("Service", service_names),
        ("PodDisruptionBudget", pdb_names),
    ):
        for name in names:
            original_uid = uid(before, kind, name)
            assert uid(after, kind, name) == original_uid, (kind, name, "upgrade UID")
            assert uid(rollback, kind, name) == original_uid, (kind, name, "rollback UID")

    for role in ROLES:
        name = f"upgrade-{role}-default"
        old = require(before, "StatefulSet", name)
        new = require(after, "StatefulSet", name)
        restored = require(rollback, "StatefulSet", name)

        assert old["metadata"]["uid"] != new["metadata"]["uid"]
        assert new["metadata"]["uid"] != restored["metadata"]["uid"]
        assert old["spec"]["serviceName"] == name
        assert new["spec"]["serviceName"] == f"{name}-headless"
        assert restored["spec"]["serviceName"] == name
        assert old["spec"]["podManagementPolicy"] == "OrderedReady"
        assert new["spec"]["podManagementPolicy"] == "Parallel"
        assert restored["spec"]["podManagementPolicy"] == "OrderedReady"

        expected_selector = {
            "trino.kubedoop.dev/cluster": "upgrade",
            "trino.kubedoop.dev/role": role,
            "trino.kubedoop.dev/role-group": "default",
        }
        assert new["spec"]["selector"]["matchLabels"] == expected_selector
        assert old["spec"]["selector"] != new["spec"]["selector"]
        assert restored["spec"]["selector"] == old["spec"]["selector"]
        assert new["metadata"]["labels"]["app.kubernetes.io/managed-by"] == (
            "operator-go"
        )
        assert new["spec"]["template"]["metadata"]["labels"][
            "app.kubernetes.io/managed-by"
        ] == "operator-go"

        old_claims = claim_specs(old)
        new_claims = claim_specs(new)
        restored_claims = claim_specs(restored)
        assert set(old_claims) == set(new_claims) == set(restored_claims) == {"data"}
        assert old_claims["data"] == new_claims["data"], (name, "upgrade claim spec")
        assert old_claims["data"] == restored_claims["data"], (name, "rollback claim spec")

        old_service_account = old["spec"]["template"]["spec"].get(
            "serviceAccountName", "default"
        )
        restored_service_account = restored["spec"]["template"]["spec"].get(
            "serviceAccountName", "default"
        )
        assert old_service_account in ("", "default")
        assert new["spec"]["template"]["spec"]["serviceAccountName"] == (
            "trinocluster-upgrade"
        )
        assert restored_service_account == old_service_account

        pdb = require(after, "PodDisruptionBudget", f"upgrade-{role}")
        selector = pdb["spec"]["selector"]["matchLabels"]
        assert selector["trino.kubedoop.dev/cluster"] == "upgrade"
        assert selector["trino.kubedoop.dev/role"] == role

    # stopped=true scales in place. Every managed identity survives stop and resume.
    stop_stable_resources = [
        *(("StatefulSet", name) for name in workload_names),
        *(("ConfigMap", name) for name in config_map_names),
        *(("Service", name) for name in service_names),
        *(("Service", f"{name}-headless") for name in workload_names),
        *(("PodDisruptionBudget", name) for name in pdb_names),
        ("ServiceAccount", "trinocluster-upgrade"),
        *(("PersistentVolumeClaim", name) for name in pvc_names),
    ]
    for kind, name in stop_stable_resources:
        expected_uid = uid(after, kind, name)
        assert uid(stopped, kind, name) == expected_uid, (kind, name, "stopped UID")
        assert uid(resumed, kind, name) == expected_uid, (kind, name, "resumed UID")

    for name in workload_names:
        assert require(after, "StatefulSet", name)["spec"]["replicas"] == 1
        assert require(stopped, "StatefulSet", name)["spec"]["replicas"] == 0
        assert require(resumed, "StatefulSet", name)["spec"]["replicas"] == 1

    expected_pvc_identity = pvc_identities(root, "before")
    for phase in ("after", "stopped", "resumed", "rollback"):
        assert pvc_identities(root, phase) == expected_pvc_identity, (
            phase,
            "PVC/PV identity changed",
        )

    for phase in ("before", "after", "resumed", "rollback"):
        result = query_result(root, phase)
        assert result == {"activeWorkers": 1, "selectOne": 1}, (phase, result)

    for phase, items in phase_resources.items():
        normalized = [normalize(items[key]) for key in sorted(items)]
        (root / f"{phase}-normalized.json").write_text(
            json.dumps(normalized, indent=2, sort_keys=True) + "\n"
        )

    for phase in ("after", "stopped", "resumed", "rollback"):
        diff = difflib.unified_diff(
            (root / "before-normalized.json").read_text().splitlines(True),
            (root / f"{phase}-normalized.json").read_text().splitlines(True),
            fromfile="before",
            tofile=phase,
        )
        (root / f"{phase}.diff").write_text("".join(diff))

    print(
        "Trino migration resource contracts passed; normalized diffs saved for review"
    )


if __name__ == "__main__":
    main(pathlib.Path(sys.argv[1]))

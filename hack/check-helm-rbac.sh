#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
helm_bin="${HELM:-helm}"
chart_dir="${repo_root}/deploy/helm/trino-operator"
manager_role="${repo_root}/config/rbac/role.yaml"
temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

rendered_role="${temp_dir}/helm-clusterrole.yaml"
manager_rules="${temp_dir}/manager-role.rules"
helm_rules="${temp_dir}/helm-clusterrole.rules"

"${helm_bin}" template trino-operator "${chart_dir}" \
  --show-only templates/clusterrole.yaml >"${rendered_role}"

extract_rules() {
  awk 'found || /^rules:$/ { found = 1; print }' "$1"
}

extract_rules "${manager_role}" >"${manager_rules}"
extract_rules "${rendered_role}" >"${helm_rules}"

if ! diff -u \
  --label config/rbac/role.yaml \
  --label deploy/helm/trino-operator/templates/clusterrole.yaml \
  "${manager_rules}" "${helm_rules}"; then
  echo "Helm ClusterRole rules differ from config/rbac/role.yaml." >&2
  exit 1
fi

echo "Helm ClusterRole rules match config/rbac/role.yaml."

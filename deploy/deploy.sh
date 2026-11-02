#!/usr/bin/env bash
# Deploy SynAgent to the k3s node over SSM. Every step is idempotent: running this
# twice lands the same state as running it once.
#
#   INSTANCE=i-... deploy/deploy.sh <registry>/synagent@sha256:... [--dry-run]
#
# Prerequisite, once: deploy/bootstrap-aws.sh
set -euo pipefail
IMAGE="${1:?usage: deploy.sh <registry/synagent@sha256:...> [--dry-run]}"
DRY="${2:-}"
[[ "$IMAGE" == *@sha256:* ]] || { echo "pin the image by digest, not a tag" >&2; exit 2; }
HERE="$(cd "$(dirname "$0")" && pwd)"
export INSTANCE="${INSTANCE:?set INSTANCE}"

OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT
for f in "$HERE"/k8s/*.yaml; do
  # A separator between files: plain concatenation merges the last document of one
  # file into the first of the next.
  printf -- '---\n'
  sed -E "s#^([[:space:]]*image:[[:space:]]*)SYN_IMAGE[[:space:]]*\$#\1${IMAGE}#" "$f" | tr -d '\r'
  printf '\n'
done > "$OUT"
grep -Eq "image:[[:space:]]*SYN_IMAGE" "$OUT" && { echo "unsubstituted SYN_IMAGE" >&2; exit 1; }
SUM=$(sha256sum "$OUT" | cut -d' ' -f1)
PAYLOAD=$(gzip -9 < "$OUT" | base64 | tr -d '\n')
echo "$(grep -c '^---' "$OUT") documents, sha256 ${SUM:0:16}"

APPLY="k3s kubectl apply -f /opt/synagent/synagent.yaml"
[ "$DRY" = "--dry-run" ] && APPLY="k3s kubectl apply --dry-run=server -f /opt/synagent/synagent.yaml"

"$HERE/ssm.sh" "$(cat <<REMOTE
set -euo pipefail
K="k3s kubectl"
mkdir -p /opt/synagent
echo '${PAYLOAD}' | base64 -d | gunzip > /opt/synagent/synagent.yaml
got=\$(sha256sum /opt/synagent/synagent.yaml | cut -d' ' -f1)
[ "\$got" = "${SUM}" ] || { echo "checksum mismatch: manifest truncated in transit" >&2; exit 1; }
echo "checksum verified on node"

if [ "${DRY}" != "--dry-run" ]; then
  # The shared mail relay admits this app by namespace and label: an additive patch,
  # skipped when the rule is already there.
  PEER='{"namespaceSelector":{"matchLabels":{"kubernetes.io/metadata.name":"synagent"}},"podSelector":{"matchLabels":{"app.kubernetes.io/part-of":"synagent"}}}'
  if \$K -n heros get networkpolicy mail -o json | grep -q '"kubernetes.io/metadata.name":"synagent"'; then
    echo "mail policy already admits synagent"
  else
    \$K -n heros patch networkpolicy mail --type=json -p "[{\"op\":\"add\",\"path\":\"/spec/ingress/-\",\"value\":{\"from\":[\$PEER],\"ports\":[{\"port\":587,\"protocol\":\"TCP\"}]}}]"
  fi
fi

${APPLY}
if [ "${DRY}" != "--dry-run" ]; then
  \$K -n synagent rollout status statefulset/postgres --timeout=300s
  \$K -n synagent rollout status statefulset/mongodb --timeout=300s
  \$K -n synagent rollout status deploy/synagent-web --timeout=240s
  \$K -n synagent rollout status deploy/synagent-worker --timeout=240s
  \$K -n synagent get pods,svc,ingress,certificate,externalsecret
fi
REMOTE
)"
echo done

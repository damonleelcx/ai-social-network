#!/usr/bin/env bash
# Build and deploy the China copy (Alibaba Cloud ECS, Hangzhou, suki.zhiyuwujie.top).
#
#   CN_HOST=root@121.43.114.65 CN_KEY=~/.ssh/suki_cn deploy/cn/deploy.sh
#
# Docker Hub and ECR are slow or blocked from mainland China, so the amd64 image is
# built here and streamed to the node's containerd over SSH. Secrets are created on
# the node separately and never appear in these manifests.
set -euo pipefail
cd "$(dirname "$0")/../.."
HOST=${CN_HOST:-root@121.43.114.65}
KEY=${CN_KEY:-$HOME/.ssh/suki_cn}
SSH=(ssh -i "$KEY" -o BatchMode=yes "$HOST")

[ -z "$(git status --porcelain -- cmd internal go.mod go.sum Dockerfile)" ] || { echo "commit first: the image is tagged with HEAD" >&2; exit 1; }
TAG=cn-$(git rev-parse --short HEAD)-amd64
IMAGE=docker.io/library/synagent:$TAG

if ! "${SSH[@]}" "k3s ctr images ls -q | grep -qx '$IMAGE'"; then
  # The Dockerfile defaults TARGETARCH to arm64 for the US node; this node is x86.
  docker buildx build --platform linux/amd64 --build-arg TARGETARCH=amd64 --provenance=false --sbom=false -t "synagent:$TAG" --load .
  docker save "synagent:$TAG" | gzip -1 | "${SSH[@]}" 'gunzip | k3s ctr images import - >/dev/null'
fi
echo "image $IMAGE"

OUT=$(mktemp)
trap 'rm -f "$OUT"' EXIT
for f in deploy/k8s/00-namespace.yaml deploy/cn/20-config.yaml deploy/k8s/30-databases.yaml deploy/cn/40-app.yaml deploy/cn/50-caddy.yaml; do
  printf -- '---\n'
  sed -E "s#^([[:space:]]*image:[[:space:]]*)SYN_IMAGE[[:space:]]*\$#\1${IMAGE}#" "$f" | tr -d '\r'
  printf '\n'
done > "$OUT"
grep -Eq "image:[[:space:]]*SYN_IMAGE" "$OUT" && { echo "unsubstituted SYN_IMAGE" >&2; exit 1; }

"${SSH[@]}" 'set -e; K="k3s kubectl"; $K apply -f - ;
  $K -n synagent rollout status statefulset/postgres --timeout=300s;
  $K -n synagent rollout status statefulset/mongodb --timeout=300s;
  $K -n synagent rollout status deploy/synagent-web --timeout=300s;
  $K -n synagent rollout status deploy/synagent-worker --timeout=300s;
  $K -n synagent rollout status deploy/caddy --timeout=120s;
  $K -n synagent get pods' < "$OUT"
echo done

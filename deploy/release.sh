#!/usr/bin/env bash
# Build the linux/arm64 image, push it to ECR by an immutable tag, and deploy it
# pinned by digest.   INSTANCE=i-... deploy/release.sh [--dry-run]
set -euo pipefail
cd "$(dirname "$0")/.."
REGION=${REGION:-us-east-1}
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
REG="$ACCOUNT.dkr.ecr.$REGION.amazonaws.com"
[ -z "$(git status --porcelain -- cmd internal go.mod go.sum Dockerfile deploy)" ] || { echo "commit first: the image is tagged with HEAD" >&2; exit 1; }
TAG=$(git rev-parse --short HEAD)
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REG" >/dev/null
if ! aws ecr describe-images --repository-name synagent --image-ids imageTag="$TAG" --region "$REGION" >/dev/null 2>&1; then
  docker buildx build --platform linux/arm64 --provenance=false --sbom=false -t "$REG/synagent:$TAG" --push .
fi
DIGEST=$(aws ecr describe-images --repository-name synagent --image-ids imageTag="$TAG" --region "$REGION" --query 'imageDetails[0].imageDigest' --output text)
echo "image $REG/synagent@$DIGEST ($TAG)"
exec deploy/deploy.sh "$REG/synagent@$DIGEST" "${1:-}"

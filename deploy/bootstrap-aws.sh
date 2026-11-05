#!/usr/bin/env bash
# One-time AWS setup for SynAgent. Safe to re-run: everything is create-if-missing,
# and existing passwords are kept. Prints no secret.
set -euo pipefail
REGION=us-east-1
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
PHOTOS=synagent-photos-$ACCOUNT
BACKUPS=synagent-backups-$ACCOUNT
HOST=match.heros-agent.space
ZONE=${ZONE:?set ZONE to the Route53 hosted zone id}
NODE_IP=${NODE_IP:?set NODE_IP to the node public IP}
ENV_FILE=${ENV_FILE:-../.env}

for b in "$PHOTOS" "$BACKUPS"; do
  aws s3api head-bucket --bucket "$b" 2>/dev/null || aws s3api create-bucket --bucket "$b" --region "$REGION" >/dev/null
  aws s3api put-public-access-block --bucket "$b" --public-access-block-configuration \
    BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true
  aws s3api put-bucket-encryption --bucket "$b" --server-side-encryption-configuration \
    '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'
  aws s3api put-bucket-ownership-controls --bucket "$b" --ownership-controls '{"Rules":[{"ObjectOwnership":"BucketOwnerEnforced"}]}'
done
aws s3api put-bucket-lifecycle-configuration --bucket "$BACKUPS" --lifecycle-configuration \
  '{"Rules":[{"ID":"expire-30d","Status":"Enabled","Filter":{"Prefix":""},"Expiration":{"Days":30}}]}'
echo "buckets ready: $PHOTOS $BACKUPS"

aws ecr describe-repositories --repository-names synagent --region "$REGION" >/dev/null 2>&1 \
  || aws ecr create-repository --repository-name synagent --region "$REGION" --image-scanning-configuration scanOnPush=true >/dev/null
echo "ecr ready"

# The node role reads only this app's secret and only these buckets. Backups can
# write but not delete; expiry is the bucket's lifecycle rule.
POLICY=$(cat <<JSON
{"Version":"2012-10-17","Statement":[
{"Effect":"Allow","Action":["secretsmanager:GetSecretValue","secretsmanager:DescribeSecret"],"Resource":"arn:aws:secretsmanager:$REGION:$ACCOUNT:secret:synagent/*"},
{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject","s3:DeleteObject"],"Resource":"arn:aws:s3:::$PHOTOS/*"},
{"Effect":"Allow","Action":["s3:ListBucket"],"Resource":["arn:aws:s3:::$PHOTOS","arn:aws:s3:::$BACKUPS"]},
{"Effect":"Allow","Action":["s3:PutObject"],"Resource":"arn:aws:s3:::$BACKUPS/*"}]}
JSON
)
aws iam put-role-policy --role-name heros-vm --policy-name SynagentAccess --policy-document "$POLICY"
echo "iam ready"

# Secrets: the model key comes from FORGE_LLM_API_KEY in ENV_FILE.
"$(command -v python || command -v python3)" "$(dirname "$0")/write_secret.py" "$ENV_FILE" "$REGION"

aws route53 change-resource-record-sets --hosted-zone-id "$ZONE" --change-batch \
  "{\"Changes\":[{\"Action\":\"UPSERT\",\"ResourceRecordSet\":{\"Name\":\"$HOST\",\"Type\":\"A\",\"TTL\":300,\"ResourceRecords\":[{\"Value\":\"$NODE_IP\"}]}}]}" >/dev/null
echo "dns ready: $HOST -> $NODE_IP"

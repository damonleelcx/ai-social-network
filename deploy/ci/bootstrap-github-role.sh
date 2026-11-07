#!/usr/bin/env bash
# Lets GitHub Actions deploy, with no stored AWS keys: main in this repository can assume a role
# that can push this one image and run commands on this one instance. Run it once, signed in to
# AWS and to gh. Safe to run again.
#   INSTANCE=i-... deploy/ci/bootstrap-github-role.sh
set -euo pipefail
INSTANCE=${INSTANCE:?set INSTANCE (the k3s node instance id)}
REGION=${REGION:-us-east-1}
REPO=${REPO:-damonleelcx/ai-social-network}
ROLE=synagent-github-deploy
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
PROVIDER=arn:aws:iam::$ACCOUNT:oidc-provider/token.actions.githubusercontent.com
# GitHub may issue subjects with immutable owner and repository ids, so ask it for the prefix.
SUB=$(gh api "repos/$REPO/actions/oidc/customization/sub" --jq '.sub_claim_prefix // empty')
SUB=${SUB:-repo:$REPO}

aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$PROVIDER" >/dev/null 2>&1 ||
  aws iam create-open-id-connect-provider --url https://token.actions.githubusercontent.com \
    --client-id-list sts.amazonaws.com --thumbprint-list ffffffffffffffffffffffffffffffffffffffff >/dev/null
echo "oidc provider ready"

TRUST=$(cat <<JSON
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Federated":"$PROVIDER"},"Action":"sts:AssumeRoleWithWebIdentity","Condition":{"StringEquals":{"token.actions.githubusercontent.com:aud":"sts.amazonaws.com","token.actions.githubusercontent.com:sub":"$SUB:ref:refs/heads/main"}}}]}
JSON
)
if aws iam get-role --role-name "$ROLE" >/dev/null 2>&1; then
  aws iam update-assume-role-policy --role-name "$ROLE" --policy-document "$TRUST"
else
  aws iam create-role --role-name "$ROLE" --assume-role-policy-document "$TRUST" >/dev/null
fi

POLICY=$(cat <<JSON
{"Version":"2012-10-17","Statement":[
{"Effect":"Allow","Action":"ecr:GetAuthorizationToken","Resource":"*"},
{"Effect":"Allow","Action":["ecr:BatchCheckLayerAvailability","ecr:BatchGetImage","ecr:CompleteLayerUpload","ecr:DescribeImages","ecr:GetDownloadUrlForLayer","ecr:InitiateLayerUpload","ecr:PutImage","ecr:UploadLayerPart"],"Resource":"arn:aws:ecr:$REGION:$ACCOUNT:repository/synagent"},
{"Effect":"Allow","Action":"ssm:SendCommand","Resource":["arn:aws:ec2:$REGION:$ACCOUNT:instance/$INSTANCE","arn:aws:ssm:$REGION::document/AWS-RunShellScript"]},
{"Effect":"Allow","Action":"ssm:GetCommandInvocation","Resource":"*"}
]}
JSON
)
aws iam put-role-policy --role-name "$ROLE" --policy-name deploy --policy-document "$POLICY"
echo "role ready"

gh variable set AWS_DEPLOY_ROLE_ARN --repo "$REPO" --body "arn:aws:iam::$ACCOUNT:role/$ROLE"
gh variable set SYN_INSTANCE --repo "$REPO" --body "$INSTANCE"
echo "variables set"

#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
LAB_DIR="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
AWS_REGION="${AWS_REGION:-$(aws configure get region)}"
POSTS_TABLE="author_post"
PROGRESS_TABLE="FanoutProgress"
API_NAME="blog-http-api"
IMAGE_TAG="${IMAGE_TAG:-git-$(git -C "$LAB_DIR" rev-parse --short HEAD)}"

[[ -n "$AWS_REGION" ]] || { echo "Configura una región con aws configure set region <region>." >&2; exit 1; }
aws_cmd() { aws --region "$AWS_REGION" "$@"; }
preflight() { echo "Región: $AWS_REGION"; echo "Sesión AWS CLI: $(aws_cmd sts get-caller-identity --output json)"; }
confirm() { local answer; read -r -p "Esto ejecutará '$1' en la cuenta mostrada. Escribe SI para continuar: " answer; [[ "$answer" == SI ]] || { echo "Cancelado."; exit 0; }; }
account_id() { aws_cmd sts get-caller-identity --query Account --output text; }
ensure_ecr() { aws_cmd ecr describe-repositories --repository-names "$1" >/dev/null 2>&1 || aws_cmd ecr create-repository --repository-name "$1" --image-tag-mutability IMMUTABLE --image-scanning-configuration scanOnPush=true >/dev/null; }
ensure_role() {
  local role="$1" policy="$2" trust
  trust="$(mktemp)"; trap 'rm -f "$trust"' RETURN
  cat >"$trust" <<'JSON'
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}
JSON
  if ! aws iam get-role --role-name "$role" >/dev/null 2>&1; then
    aws iam create-role --role-name "$role" --assume-role-policy-document "file://$trust" >/dev/null
    aws iam attach-role-policy --role-name "$role" --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
  fi
  aws iam put-role-policy --role-name "$role" --policy-name LabPermissions --policy-document "$policy"
  rm -f "$trust"; trap - RETURN
}

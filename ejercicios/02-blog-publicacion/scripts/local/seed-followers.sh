#!/usr/bin/env bash
set -euo pipefail

# Inserta followers ficticios en DynamoDB Local para probar el fanout.
# Usa las mismas PK/SK de laboratorio y sobrescribe exactamente esos items.
TABLE_NAME="${TABLE_NAME:-author_post}"
AWS_REGION="${AWS_REGION:-us-east-1}"
DYNAMODB_ENDPOINT="${DYNAMODB_ENDPOINT:-http://localhost:8000}"
AUTHOR_ID="${AUTHOR_ID:-author-1}"
FOLLOWER_COUNT="${FOLLOWER_COUNT:-10}"

[[ "$FOLLOWER_COUNT" =~ ^[1-9][0-9]*$ ]] || { echo "FOLLOWER_COUNT debe ser entero positivo" >&2; exit 1; }
aws_args=(--region "$AWS_REGION" --endpoint-url "$DYNAMODB_ENDPOINT")
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

for ((number=1; number<=FOLLOWER_COUNT; number++)); do
  printf -v follower_id 'user-%03d' "$number"
  aws dynamodb put-item --table-name "$TABLE_NAME" "${aws_args[@]}" --item "{\"PK\":{\"S\":\"FOLLOWING#${AUTHOR_ID}\"},\"SK\":{\"S\":\"FOLLOWER#${follower_id}\"},\"follower_id\":{\"S\":\"${follower_id}\"},\"following_id\":{\"S\":\"${AUTHOR_ID}\"}}"
done

echo "Seed terminado: ${FOLLOWER_COUNT} followers para ${AUTHOR_ID} en ${TABLE_NAME}."

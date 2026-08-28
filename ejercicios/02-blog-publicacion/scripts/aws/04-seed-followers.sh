#!/usr/bin/env bash
source "$(cd -- "$(dirname -- "$0")" && pwd)/_common.sh"
preflight; confirm "insertar 250 followers DEMO en DynamoDB real"
for ((number=1; number<=250; number++)); do printf -v id 'user-%03d' "$number"; aws_cmd dynamodb put-item --table-name "$POSTS_TABLE" --item "{\"PK\":{\"S\":\"FOLLOWING#author-1\"},\"SK\":{\"S\":\"FOLLOWER#$id\"},\"follower_id\":{\"S\":\"$id\"},\"following_id\":{\"S\":\"author-1\"}}"; done
echo "Seed terminado: 250 followers para author-1."

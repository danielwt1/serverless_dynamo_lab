#!/usr/bin/env bash
source "$(cd -- "$(dirname -- "$0")" && pwd)/_common.sh"
preflight; command -v docker >/dev/null || { echo "Docker es necesario." >&2; exit 1; }; confirm "construir y subir imágenes Lambda a ECR"
ACCOUNT_ID="$(account_id)"; REGISTRY="$ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"; aws_cmd ecr get-login-password | docker login --username AWS --password-stdin "$REGISTRY"
for item in "lambda_create_post:blog-create-post" "lambda_get_posts:blog-get-posts" "fanout_starter:blog-fanout-starter" "follower_fanout_worker:blog-follower-worker"; do dir="${item%%:*}"; repository="${item##*:}"; image="$REGISTRY/$repository:$IMAGE_TAG"; docker buildx build --platform linux/arm64 --provenance=false --load -t "$image" "$LAB_DIR/02-lambda/functions/$dir"; docker push "$image"; done
echo "Imágenes subidas con tag inmutable: $IMAGE_TAG"

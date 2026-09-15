#!/usr/bin/env bash
set -euo pipefail

exercise_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

verify_go_module() {
  local module="$1"
  local module_dir="${exercise_dir}/02-lambda/functions/${module}"

  if command -v go >/dev/null 2>&1; then
    (cd "${module_dir}" && go test ./... && go vet ./...)
    return
  fi

  if ! command -v docker >/dev/null 2>&1; then
    echo "Se requiere Go 1.27 o Docker para verificar ${module}" >&2
    return 1
  fi

  docker run --rm \
    -v "${module_dir}:/workspace" \
    -w /workspace \
    --entrypoint /bin/sh \
    "${GO_TEST_IMAGE:-golang:1.27.1-alpine}" \
    -c 'go test ./... && go vet ./...'
}

for module in get_tasks_querys lambda_create_task lambda_complete_task lambda_find_expired_task lambda_task_overdue_consumer; do
  echo "Verificando ${module}"
  verify_go_module "${module}"
done
(cd "${exercise_dir}/04-cdk" && npm test && npm run synth)

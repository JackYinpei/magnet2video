#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="$repo_root/deploy/docker-compose.worker.yml"

worker_env_name="${WORKER_ENV_FILE:-.env.worker}"
if [[ "$worker_env_name" = /* ]]; then
  env_file="$worker_env_name"
  compose_env_file="$worker_env_name"
else
  env_file="$repo_root/$worker_env_name"
  compose_env_file="$worker_env_name"
fi

compose() {
  WORKER_ENV_FILE="$compose_env_file" docker compose --project-directory "$repo_root" -f "$compose_file" "$@"
}

get_env_value() {
  local key="$1"
  awk -F= -v key="$key" '
    $1 == key {
      value = substr($0, length(key) + 2)
      sub(/[[:space:]]+#.*$/, "", value)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      print value
      exit
    }
  ' "$env_file"
}

usage() {
  cat <<'USAGE'
Usage: scripts/run-worker-local.sh [command]

Commands:
  up       Pull image, start local worker, then follow logs (default)
  start    Start local worker in background
  pull     Pull latest worker image
  logs     Follow worker logs
  ps       Show worker container status
  restart  Restart local worker
  stop     Stop local worker
  down     Remove local worker container/network

Requires repo-root .env.worker with:
  WORKER_ID=home-worker-01
  RABBITMQ_URL=amqp://worker:<pass>@<server-public-ip>:5672/magnet
  S3_* / CLOUD_STORAGE_* matching the server

Set WORKER_ENV_FILE=.env to use another env file.
USAGE
}

preflight() {
  if [[ ! -f "$env_file" ]]; then
    echo "Missing $env_file. Copy .env.example to $worker_env_name and fill worker settings." >&2
    exit 1
  fi

  if [[ ! -f "$compose_file" ]]; then
    echo "Missing $compose_file." >&2
    exit 1
  fi

  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is not installed or not in PATH." >&2
    exit 1
  fi

  if ! docker compose version >/dev/null 2>&1; then
    echo "Docker Compose v2 is required: docker compose ..." >&2
    exit 1
  fi

  local rabbitmq_url
  rabbitmq_url="$(get_env_value RABBITMQ_URL)"
  if [[ -z "$rabbitmq_url" ]]; then
    echo "RABBITMQ_URL is empty in $env_file." >&2
    exit 1
  fi

  if [[ "$rabbitmq_url" == *"@rabbitmq:"* ]]; then
    echo "RABBITMQ_URL points to '@rabbitmq:', which only works inside the server compose network." >&2
    echo "For a local worker, use the server public IP, for example:" >&2
    echo "  RABBITMQ_URL=amqp://worker:<pass>@<server-public-ip>:5672/magnet" >&2
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    echo "Docker daemon is not running. Start Docker Desktop, then rerun this script." >&2
    exit 1
  fi

  mkdir -p "$repo_root/.logs" "$repo_root/download"
}

cmd="${1:-up}"

case "$cmd" in
  -h|--help|help)
    usage
    ;;
  up)
    preflight
    compose pull app
    compose up -d
    compose logs -f app
    ;;
  start)
    preflight
    compose up -d
    ;;
  pull)
    preflight
    compose pull app
    ;;
  logs)
    preflight
    compose logs -f app
    ;;
  ps)
    preflight
    compose ps
    ;;
  restart)
    preflight
    compose restart app
    compose logs -f app
    ;;
  stop)
    preflight
    compose stop app
    ;;
  down)
    preflight
    compose down
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac

#!/bin/sh
set -eu

client_name="${1:-}"
image="${OPENCHAT_CLIENT_IMAGE:-openchat-wx:local}"
ui_port="${OPENCHAT_CLIENT_UI_PORT:-9001}"

if [ -z "$client_name" ]; then
  clients=$(docker ps --filter 'name=^/client_' --format '{{.Names}}')
  count=$(printf '%s\n' "$clients" | sed '/^$/d' | wc -l | tr -d ' ')
  if [ "$count" -ne 1 ]; then
    echo "Please pass the client container name when zero or multiple clients are running." >&2
    exit 1
  fi
  client_name="$clients"
fi

if ! docker inspect "$client_name" >/dev/null 2>&1; then
  echo "Client container not found: $client_name" >&2
  exit 1
fi

backup_name="${client_name}_backup_$(date +%Y%m%d-%H%M%S)"
env_file=$(mktemp)
chmod 600 "$env_file"
trap 'rm -f "$env_file"' EXIT
docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$client_name" > "$env_file"

echo "Stopping $client_name"
docker stop "$client_name" >/dev/null
docker rename "$client_name" "$backup_name"

if docker run -d \
  --name "$client_name" \
  --restart always \
  --network wechat-robot \
  --env-file "$env_file" \
  --volumes-from "$backup_name" \
  -p "127.0.0.1:${ui_port}:9000" \
  "$image" >/dev/null; then
  echo "Started $client_name; rollback container: $backup_name"
  echo "Provider UI: http://127.0.0.1:${ui_port}/api/v1/robot/ai-providers/ui"
  exit 0
fi

echo "New client failed to start; rolling back." >&2
docker rm -f "$client_name" >/dev/null 2>&1 || true
docker rename "$backup_name" "$client_name"
docker start "$client_name" >/dev/null
exit 1

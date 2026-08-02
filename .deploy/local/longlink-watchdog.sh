#!/bin/sh
set -eu

check_interval="${CHECK_INTERVAL_SECONDS:-30}"
failure_threshold="${FAILURE_THRESHOLD:-3}"
proxy_container="${PROXY_CONTAINER:-wechat-longlink-proxy}"
state_dir=/tmp/openchat-longlink-watchdog
mkdir -p "$state_dir"

echo "[longlink-watchdog] started: interval=${check_interval}s threshold=${failure_threshold}"

while true; do
  for server in $(docker ps --filter 'name=^/server_' --format '{{.Names}}'); do
    ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$server" 2>/dev/null || true)
    [ -n "$ip" ] || continue

    # Do not touch a server before it has established its first long link. This
    # avoids restart loops while a new robot is still waiting for QR login.
    if ! docker logs "$proxy_container" 2>&1 | grep -Fq "connection from AF=2 ${ip}:"; then
      continue
    fi

    key=$(printf '%s' "$server" | tr -c 'A-Za-z0-9_.-' '_')
    state_file="$state_dir/$key"
    if docker exec "$proxy_container" sh -c "netstat -tn 2>/dev/null | grep -F '$ip:' | grep -q 'ESTABLISHED'"; then
      printf '0' > "$state_file"
      continue
    fi

    misses=0
    [ ! -f "$state_file" ] || misses=$(cat "$state_file")
    misses=$((misses + 1))
    printf '%s' "$misses" > "$state_file"
    echo "[longlink-watchdog] $server has no established long link ($misses/$failure_threshold)"

    if [ "$misses" -ge "$failure_threshold" ]; then
      echo "[longlink-watchdog] restarting $server to recover inbound messages"
      if docker restart "$server" >/dev/null; then
        printf '0' > "$state_file"
      fi
    fi
  done
  sleep "$check_interval"
done

#!/bin/sh
set -e

# Start config watcher in background
# Watches /etc/tengine/conf.d and /etc/tengine/default.d for changes and reloads tengine
(
  while true; do
    inotifywait -q -r -e modify,create,delete,move /etc/tengine/conf.d /etc/tengine/default.d 2>/dev/null || sleep 5
    sleep 1
    if tengine -t 2>/dev/null; then
      echo "[watcher] Config valid, reloading tengine..."
      tengine -s reload 2>/dev/null || true
    else
      echo "[watcher] Config test failed, skipping reload"
    fi
  done
) &

# Start tengine in foreground
exec tengine -g "daemon off;"

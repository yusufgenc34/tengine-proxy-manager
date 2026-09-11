#!/bin/sh
set -eu

# The backend may populate the shared volume before this container is created.
# Restore image-owned base files without replacing generated host configurations.
cp /opt/tengine-defaults/nginx.conf /opt/tengine-defaults/mime.types /etc/tengine/
mkdir -p /etc/tengine/html/.well-known/acme-challenge /var/log/tengine

# Validate explicitly requested changes and acknowledge the result to the backend.
# Never reload uncommitted partial writes merely because an inotify event fired.
mkdir -p /etc/tengine/control
chmod 700 /etc/tengine/control
(
    while true; do
        for request in /etc/tengine/control/*.request; do
            [ -f "$request" ] || continue
            base=${request%.request}
            [ ! -f "$base.response" ] || continue
            deadline=$(cat "$request" 2>/dev/null) || continue
            case "$deadline" in ''|*[!0-9]*) continue ;; esac
            if [ "$(date +%s)" -ge "$deadline" ]; then
                printf 'expired\n' > "$base.result"
            elif timeout 5 tengine -t > "$base.result" 2>&1 && [ "$(date +%s)" -lt "$deadline" ] && timeout 2 tengine -s reload >> "$base.result" 2>&1; then
                printf 'ok\n' > "$base.result"
            fi
            if [ -f "$request" ]; then
                mv "$base.result" "$base.response"
            else
                rm -f "$base.result"
            fi
        done
        sleep 0.2
    done
) &
exec tengine -g "daemon off;"

#!/bin/sh
set -e

# Auto-generate missing self-signed SSL certificates
# Scans all conf files and creates certs for any referenced paths that don't exist
echo "[entrypoint] Checking SSL certificates..."
for conf in /etc/tengine/conf.d/*.conf; do
	[ -f "$conf" ] || continue

	cert_path=$(grep 'ssl_certificate ' "$conf" 2>/dev/null | head -1 | sed -n 's/.*ssl_certificate\s\+\([^;]\+\).*/\1/p')
	key_path=$(grep 'ssl_certificate_key ' "$conf" 2>/dev/null | head -1 | sed -n 's/.*ssl_certificate_key\s\+\([^;]\+\).*/\1/p')

	if [ -n "$cert_path" ] && [ ! -f "$cert_path" ]; then
		domain=$(basename "$(dirname "$cert_path")")
		echo "[entrypoint] Missing SSL cert for '$domain', generating self-signed..."
		mkdir -p "$(dirname "$cert_path")"
		key_out="${key_path:-$(dirname "$cert_path")/privkey.pem}"
		openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
			-keyout "$key_out" \
			-out "$cert_path" \
			-subj "/CN=$domain" 2>/dev/null
		echo "[entrypoint] Generated: $cert_path"
	fi
done
echo "[entrypoint] SSL check complete."

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

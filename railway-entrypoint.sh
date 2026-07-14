#!/bin/sh
# ============================================================
# Railway-aware entrypoint wrapper for DX
# ------------------------------------------------------------
# Railway injects a dynamic $PORT env var at runtime and routes
# public traffic to whatever port your app listens on (it must
# bind 0.0.0.0:$PORT). DX does NOT read its web port from an
# env var / config file at boot -- it's stored in its database
# (sqlite/postgres) and changed via the built-in CLI:
#
#     DX setting -port <N> -listenIP <ip>
#
# So on Railway (and only on Railway, where $PORT is set) we run
# that CLI command once before starting the real entrypoint,
# to keep the panel port in sync with whatever port Railway
# assigned this deployment. On a normal docker-compose / VPS
# install $PORT is not set, so this block is skipped and nothing
# changes for existing users.
# ============================================================
set -e

if [ -n "$PORT" ]; then
    echo "[railway-entrypoint] \$PORT=$PORT detected, syncing DX panel port/listenIP..."
    /app/DX setting -port "$PORT" -listenIP 0.0.0.0 || \
        echo "[railway-entrypoint] warning: could not set port via CLI (first boot creates DB on DX start; will retry next restart if this fails)"
fi

exec /app/DockerEntrypoint.sh "$@"

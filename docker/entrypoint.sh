#!/bin/sh
# sonarr2 container entrypoint.
#
# Reads PUID/PGID env vars (defaulting to 1000/1000) and runs the binary
# as that uid:gid via su-exec. Matches the LinuxServer.io convention so
# bind-mounted /config and /data volumes inherit permissions the host
# user controls.
#
# /config is the app's own state dir; we ensure it exists and is writable
# by the target user. /data is the user's media library — we do NOT
# recursively chown it (could be terabytes). Users are responsible for
# making /data writable by PUID:PGID on the host.

set -e

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

# Guard against obviously-bad input so we fail loudly instead of chown'ing
# to a nonsensical id.
case "$PUID" in
    ''|*[!0-9]*) echo "entrypoint: PUID must be a positive integer, got '$PUID'" >&2; exit 1 ;;
esac
case "$PGID" in
    ''|*[!0-9]*) echo "entrypoint: PGID must be a positive integer, got '$PGID'" >&2; exit 1 ;;
esac

mkdir -p /config /config/data
# chown may fail on a mount whose underlying fs doesn't support it (e.g. a
# read-only ConfigMap). Treat failure as a warning and keep going — the
# user will see permission errors at first write if it's really wrong.
if ! chown "$PUID:$PGID" /config 2>/dev/null; then
    echo "entrypoint: warning: could not chown /config to $PUID:$PGID (continuing)" >&2
fi

# Auto-migrate pre-fix database location. Builds before the WORKDIR fix
# wrote the SQLite DB to /data/sonarr2.db (inside the media-library
# volume) because the default DSN was relative to the container root.
# Move it into /config/data/ on first boot of the fixed image so users
# don't have to re-populate their library.
old_db="/data/sonarr2.db"
new_dir="/config/data"
new_db="$new_dir/sonarr2.db"

if [ -f "$old_db" ] && [ ! -f "$new_db" ]; then
    echo "entrypoint: found pre-fix database at $old_db; migrating to $new_db" >&2
    if mv "$old_db" "$new_db" 2>/dev/null; then
        for ext in -shm -wal; do
            [ -f "$old_db$ext" ] && mv "$old_db$ext" "$new_db$ext" 2>/dev/null || true
        done
        chown -R "$PUID:$PGID" "$new_dir" 2>/dev/null || true
        echo "entrypoint: migration complete — old /data/sonarr2.db* files are now in /config/data/" >&2
    else
        echo "entrypoint: warning: failed to migrate $old_db (permission? read-only?); leaving in place" >&2
    fi
fi

# su-exec accepts numeric "uid:gid" — no user account needed in /etc/passwd.
exec su-exec "$PUID:$PGID" /sonarr2 "$@"

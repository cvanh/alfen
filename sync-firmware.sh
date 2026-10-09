#!/usr/bin/env bash
#
# sync-firmware.sh — log in to the FTP server, find new/updated files and
# download them into ./firmware. Re-running only fetches what changed.
#
# Usage:
#   ./sync-firmware.sh                 # sync everything under the FTP root
#   ./sync-firmware.sh Firmware        # sync only a sub-path on the server
#
set -euo pipefail

# --- Config -----------------------------------------------------------------
FTP_HOST="${FTP_HOST:-52.174.252.156}"
FTP_USER="${FTP_USER:-installer}"
FTP_PASS="${FTP_PASS:-jIf978FQmk1W}"

# Where downloads land (relative to this script).
DEST="$(cd "$(dirname "$0")" && pwd)/firmware"

# Optional sub-path on the server to limit the sync.
REMOTE_PATH="${1:-}"

# --- Sync -------------------------------------------------------------------
mkdir -p "$DEST"

log="$(mktemp)"
trap 'rm -f "$log"' EXIT

# This is --mirror spelled out (-r -N -l inf) but WITHOUT its implied
# --no-remove-listing, so wget deletes the .listing files it uses for
# timestamp comparison instead of leaving them scattered in the tree.
#   -r / -l inf    : recurse the whole tree.
#   -N             : timestamping — only pull files that are new or newer.
#   --no-verbose   : quiet, but still logs one line per file actually fetched.
#   -nH            : don't create a host-named top directory.
#   --no-parent    : stay within the requested sub-path.
set +e
wget \
  --recursive --level=inf \
  --timestamping \
  --no-host-directories \
  --no-parent \
  --no-verbose \
  --directory-prefix="$DEST" \
  --ftp-user="$FTP_USER" \
  --ftp-password="$FTP_PASS" \
  "ftp://${FTP_HOST}/${REMOTE_PATH}" 2>"$log"
status=$?
set -e

if [ "$status" -ne 0 ]; then
  echo "FTP sync failed (wget exit $status):" >&2
  cat "$log" >&2
  exit "$status"
fi

# Each downloaded file is logged as: ... -> "DEST/path" [N]
count="$(grep -c ' -> "' "$log" || true)"

if [ "$count" -eq 0 ]; then
  echo "Up to date — no new or changed files."
else
  echo "Downloaded $count new/updated file(s):"
  grep -oE ' -> "[^"]+"' "$log" | sed -E 's/ -> "(.*)"/  \1/'
fi

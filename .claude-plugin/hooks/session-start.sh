#!/usr/bin/env bash
set -euo pipefail

if ! command -v wariwari >/dev/null 2>&1; then
  cat << 'MSG'
<hook-output>
WariWari plugin active - CLI not found on PATH.
</hook-output>
MSG
  exit 0
fi

if wariwari auth status --quiet 2>/dev/null | grep -q '"authenticated": true'; then
  cat << 'MSG'
<hook-output>
WariWari plugin active.
</hook-output>
MSG
else
  cat << 'MSG'
<hook-output>
WariWari plugin active - not authenticated.
Run: wariwari setup
</hook-output>
MSG
fi


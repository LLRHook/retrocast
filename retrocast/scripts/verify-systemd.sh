#!/usr/bin/env bash
set -euo pipefail
if output=$(systemd-analyze --user verify "$@" 2>&1); then
    printf '%s\n' "$output"
else
    printf '%s\n' "$output" >&2
    exit 1
fi
# systemd can return success after ignoring invalid settings.
if printf '%s\n' "$output" | grep -Eq 'Unknown key|Failed to parse|bad unit file'; then
    exit 1
fi

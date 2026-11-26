#!/bin/sh
set -eu
# Never trace this script or print mc output; credentials stay off argv.
: "${MINIO_SIGNER_ACCESS_KEY:?required}"
: "${MINIO_SIGNER_SECRET_KEY:?required}"
: "${MINIO_ROOT_USER:?required}"
: "${MINIO_ROOT_PASSWORD:?required}"
[ "$MINIO_SIGNER_ACCESS_KEY" != "$MINIO_ROOT_USER" ] || { echo 'Signer must be a dedicated account' >&2; exit 1; }
# Backstop for the pinned MC_HOST parser and HTTP credential headers.
# PR #94 also rejects these in the shared validator before services start.
newline='
'
case "$MINIO_ROOT_USER" in *:*|*"$newline"*) echo 'MINIO_ROOT_USER contains an unsupported character; rotate MINIO_ACCESS_KEY' >&2; exit 1;; esac
if printf '%s' "$MINIO_ROOT_USER" | LC_ALL=C tr -d '\011' | LC_ALL=C grep -q '[[:cntrl:]]'; then
 echo 'MINIO_ROOT_USER contains an unsupported control character; rotate MINIO_ACCESS_KEY' >&2; exit 1
fi
case "$MINIO_ROOT_PASSWORD" in *:*|*"$newline"*) echo 'MINIO_ROOT_PASSWORD contains an unsupported character; rotate it' >&2; exit 1;; esac
trap 'echo "MinIO signer initialization failed" >&2' EXIT
# MC_HOST is parsed literally by the pinned client; do not URL-encode it.
export MC_HOST_local="http://${MINIO_ROOT_USER}:${MINIO_ROOT_PASSWORD}@minio:9000"
n=0
until mc ready local >/dev/null 2>&1; do
 n=$((n+1)); [ "$n" -lt 60 ] || exit 1
 sleep 1
done
mc mb --ignore-existing local/retrocast >/dev/null 2>&1
printf '%s\n' "$MINIO_SIGNER_SECRET_KEY" | mc admin user add local "$MINIO_SIGNER_ACCESS_KEY" >/dev/null 2>&1
mc admin policy create local retrocast-signer /init/policy.json >/dev/null 2>&1
mc admin policy attach local retrocast-signer --user "$MINIO_SIGNER_ACCESS_KEY" >/dev/null 2>&1
trap - EXIT
echo 'MinIO signer initialized'

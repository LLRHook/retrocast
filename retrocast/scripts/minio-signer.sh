#!/bin/sh
set -eu
# Never trace this script or print mc output; credentials stay off argv.
: "${MINIO_SIGNER_ACCESS_KEY:?required}"
: "${MINIO_SIGNER_SECRET_KEY:?required}"
: "${MINIO_ROOT_USER:?required}"
: "${MINIO_ROOT_PASSWORD:?required}"
[ "$MINIO_SIGNER_ACCESS_KEY" != "$MINIO_ROOT_USER" ] || { echo 'Signer must be a dedicated account' >&2; exit 1; }
trap 'echo "MinIO signer initialization failed" >&2' EXIT
# Encode through stdin-only utilities so punctuation in credentials is safe too.
encode() { printf '%s' "$1" | od -An -tx1 | tr -d ' \n' | sed 's/../%&/g'; }
export MC_HOST_local="http://$(encode "$MINIO_ROOT_USER"):$(encode "$MINIO_ROOT_PASSWORD")@minio:9000"
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

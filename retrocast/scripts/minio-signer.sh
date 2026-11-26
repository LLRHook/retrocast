#!/bin/sh
set -eu
# Do not enable shell tracing or print mc output: user add includes credentials.
: "${MINIO_SIGNER_ACCESS_KEY:?required}"
: "${MINIO_SIGNER_SECRET_KEY:?required}"
: "${MINIO_ROOT_USER:?required}"
: "${MINIO_ROOT_PASSWORD:?required}"
[ "$MINIO_SIGNER_ACCESS_KEY" != "$MINIO_ROOT_USER" ] || { echo 'Signer must be a dedicated account' >&2; exit 1; }
trap 'echo "MinIO signer initialization failed" >&2' EXIT
# Bounded startup wait; each attempt authenticates and checks server readiness.
n=0
until mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null 2>&1 && mc ready local >/dev/null 2>&1; do
 n=$((n+1)); [ "$n" -lt 60 ] || exit 1
 sleep 1
done
mc mb --ignore-existing local/retrocast >/dev/null 2>&1
mc admin user add local "$MINIO_SIGNER_ACCESS_KEY" "$MINIO_SIGNER_SECRET_KEY" >/dev/null 2>&1
mc admin policy create local retrocast-signer /init/policy.json >/dev/null 2>&1
# Deny rules bound any other grants.
mc admin policy attach local retrocast-signer --user "$MINIO_SIGNER_ACCESS_KEY" >/dev/null 2>&1
trap - EXIT
echo 'MinIO signer initialized'

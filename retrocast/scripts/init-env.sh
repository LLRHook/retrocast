#!/usr/bin/env bash
set -euo pipefail
# Values stay in this process and the private destination file; never enable tracing.
server_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
destination=${1:-"$server_dir/.env"}
if [ -e "$destination" ]; then echo 'Environment file already exists; refusing to overwrite. See docs/OPERATIONS.md for rotation.' >&2; exit 1; fi
command -v openssl >/dev/null || { echo 'openssl is required to generate secrets.' >&2; exit 1; }
umask 077
set -C
exec 3>"$destination"
set +C
trap 'rm -f -- "$destination"' EXIT
# A letter plus 32 hex characters fits every consumer and stays a YAML string.
# Public key IDs are a letter plus 19 hex characters.
jwt=s$(openssl rand -hex 16)
postgres=s$(openssl rand -hex 16)
minio=s$(openssl rand -hex 16)
signer=s$(openssl rand -hex 16)
livekit=s$(openssl rand -hex 16)
grafana=s$(openssl rand -hex 16)
minio_key=k$(openssl rand -hex 10 | cut -c1-19)
signer_key=k$(openssl rand -hex 10 | cut -c1-19)
livekit_key=k$(openssl rand -hex 10 | cut -c1-19)
while IFS= read -r line || [ -n "$line" ]; do
 line=${line%$'\r'}
 case "$line" in
 JWT_SECRET=*) line="JWT_SECRET=$jwt";;
 POSTGRES_PASSWORD=*) line="POSTGRES_PASSWORD=$postgres";;
 DATABASE_URL=*) line="DATABASE_URL=postgres://retrocast:$postgres@localhost:5432/retrocast?sslmode=disable";;
 MINIO_SECRET_KEY=*) line="MINIO_SECRET_KEY=$minio";;
 MINIO_ROOT_PASSWORD=*) line="MINIO_ROOT_PASSWORD=$minio";;
 MINIO_ACCESS_KEY=*) line="MINIO_ACCESS_KEY=$minio_key";;
 MINIO_SIGNER_SECRET_KEY=*) line="MINIO_SIGNER_SECRET_KEY=$signer";;
 MINIO_SIGNER_ACCESS_KEY=*) line="MINIO_SIGNER_ACCESS_KEY=$signer_key";;
 LIVEKIT_API_SECRET=*) line="LIVEKIT_API_SECRET=$livekit";;
 LIVEKIT_API_KEY=*) line="LIVEKIT_API_KEY=$livekit_key";;
 GRAFANA_ADMIN_PASSWORD=*) line="GRAFANA_ADMIN_PASSWORD=$grafana";;
 esac
 printf '%s\n' "$line" >&3
done < "$server_dir/.env.example"
exec 3>&-
chmod 600 "$destination"
trap - EXIT
echo 'Created private environment file. Keep it private; see docs/OPERATIONS.md before upgrading an existing data volume.'

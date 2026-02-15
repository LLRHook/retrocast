#!/usr/bin/env bash
#
# demo.sh — Seed the Retrocast database with demo data.
#
# Usage:
#   ./scripts/demo.sh
#
# Prerequisites:
#   - The Retrocast server must be running (e.g. via docker compose up)
#   - curl and jq must be installed
#
# This script creates:
#   - 2 users: alice and bob (password: password123)
#   - 1 guild: "Retrocast Demo Server" (owned by alice)
#   - 3 text channels: #general, #random, #announcements
#   - Bob joins the guild via invite
#   - A "Moderator" role assigned to alice
#   - Sample messages in #general from both users

set -euo pipefail

BASE_URL="${RETROCAST_URL:-http://localhost:8080}"
API="$BASE_URL/api/v1"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

die() {
    echo "FATAL: $1" >&2
    exit 1
}

# Make an API call, check HTTP status, and return the response body.
# Usage: api_call METHOD PATH [DATA]
# Sets global $BODY with the response.
api_call() {
    local method="$1" path="$2" data="${3:-}" token="${4:-}"
    local url="$API$path"
    local http_code

    local -a curl_args=(
        -s -w '\n%{http_code}'
        -X "$method"
        -H 'Content-Type: application/json'
    )

    if [[ -n "$token" ]]; then
        curl_args+=(-H "Authorization: Bearer $token")
    fi

    if [[ -n "$data" ]]; then
        curl_args+=(-d "$data")
    fi

    local response
    response=$(curl "${curl_args[@]}" "$url") || die "curl failed for $method $path"

    http_code=$(echo "$response" | tail -n1)
    BODY=$(echo "$response" | sed '$d')

    # 2xx is success
    if [[ "$http_code" -lt 200 || "$http_code" -ge 300 ]]; then
        # Return non-zero so callers can handle gracefully
        return 1
    fi
    return 0
}

# Extract a JSON field using jq.
json_field() {
    echo "$BODY" | jq -r "$1"
}

# ---------------------------------------------------------------------------
# Wait for server
# ---------------------------------------------------------------------------

echo "=> Waiting for server at $BASE_URL ..."
for i in $(seq 1 30); do
    if curl -sf "$BASE_URL/health" > /dev/null 2>&1; then
        echo "   Server is ready."
        break
    fi
    if [[ "$i" -eq 30 ]]; then
        die "Server not reachable at $BASE_URL after 30 seconds."
    fi
    sleep 1
done

# ---------------------------------------------------------------------------
# Step 1: Register users
# ---------------------------------------------------------------------------

echo "=> Registering user alice ..."
if api_call POST "/auth/register" '{"username":"alice","password":"password123"}'; then
    ALICE_TOKEN=$(json_field '.access_token')
    ALICE_ID=$(json_field '.user.id')
    echo "   Created alice (id=$ALICE_ID)"
else
    echo "   alice may already exist, logging in ..."
    api_call POST "/auth/login" '{"username":"alice","password":"password123"}' || die "Cannot register or log in as alice"
    ALICE_TOKEN=$(json_field '.access_token')
    ALICE_ID=$(json_field '.user.id')
    echo "   Logged in as alice (id=$ALICE_ID)"
fi

echo "=> Registering user bob ..."
if api_call POST "/auth/register" '{"username":"bob","password":"password123"}'; then
    BOB_TOKEN=$(json_field '.access_token')
    BOB_ID=$(json_field '.user.id')
    echo "   Created bob (id=$BOB_ID)"
else
    echo "   bob may already exist, logging in ..."
    api_call POST "/auth/login" '{"username":"bob","password":"password123"}' || die "Cannot register or log in as bob"
    BOB_TOKEN=$(json_field '.access_token')
    BOB_ID=$(json_field '.user.id')
    echo "   Logged in as bob (id=$BOB_ID)"
fi

# ---------------------------------------------------------------------------
# Step 2: Create guild
# ---------------------------------------------------------------------------

echo "=> Creating guild 'Retrocast Demo Server' ..."
if api_call POST "/guilds" '{"name":"Retrocast Demo Server"}' "$ALICE_TOKEN"; then
    GUILD_ID=$(json_field '.data.id')
    echo "   Created guild (id=$GUILD_ID)"
else
    # Check if alice already has the guild
    echo "   Guild creation failed, checking existing guilds ..."
    api_call GET "/users/@me/guilds" "" "$ALICE_TOKEN" || die "Cannot list guilds"
    GUILD_ID=$(echo "$BODY" | jq -r '.data[] | select(.name == "Retrocast Demo Server") | .id' | head -1)
    if [[ -z "$GUILD_ID" || "$GUILD_ID" == "null" ]]; then
        die "Failed to create or find the demo guild."
    fi
    echo "   Found existing guild (id=$GUILD_ID)"
fi

# ---------------------------------------------------------------------------
# Step 3: Create channels
# ---------------------------------------------------------------------------

# Helper: create a text channel if it doesn't already exist.
create_channel() {
    local name="$1"
    echo "=> Creating channel #$name ..."

    if api_call POST "/guilds/$GUILD_ID/channels" \
        "{\"name\":\"$name\",\"type\":0}" "$ALICE_TOKEN"; then
        local ch_id
        ch_id=$(json_field '.data.id')
        echo "   Created #$name (id=$ch_id)"
        echo "$ch_id"
        return
    fi

    # Channel might exist; list and find it.
    echo "   Channel creation failed, checking existing channels ..."
    api_call GET "/guilds/$GUILD_ID/channels" "" "$ALICE_TOKEN" || die "Cannot list channels"
    local ch_id
    ch_id=$(echo "$BODY" | jq -r ".data[] | select(.name == \"$name\") | .id" | head -1)
    if [[ -z "$ch_id" || "$ch_id" == "null" ]]; then
        die "Failed to create or find channel #$name"
    fi
    echo "   Found existing #$name (id=$ch_id)"
    echo "$ch_id"
}

# Capture channel IDs (last line of create_channel output is the ID).
GENERAL_ID=$(create_channel "general" | tail -1)
RANDOM_ID=$(create_channel "random" | tail -1)
ANNOUNCEMENTS_ID=$(create_channel "announcements" | tail -1)

# ---------------------------------------------------------------------------
# Step 4: Create invite and have bob join
# ---------------------------------------------------------------------------

echo "=> Creating invite for guild ..."
if api_call POST "/guilds/$GUILD_ID/invites" '{"max_uses":10,"max_age_seconds":86400}' "$ALICE_TOKEN"; then
    INVITE_CODE=$(json_field '.code')
    echo "   Invite code: $INVITE_CODE"
else
    die "Failed to create invite."
fi

echo "=> Bob joining guild via invite ..."
if api_call POST "/invites/$INVITE_CODE" '' "$BOB_TOKEN"; then
    echo "   Bob joined the guild."
else
    echo "   Bob may already be a member (skipping)."
fi

# ---------------------------------------------------------------------------
# Step 5: Create Moderator role and assign to alice
# ---------------------------------------------------------------------------

# Moderator permissions: ViewChannel(1) | SendMessages(2) | ManageMessages(4) |
# KickMembers(32) | MentionEveryone(8192) | ReadMessageHistory(32768) | CreateInvite(65536)
MOD_PERMS=$((1 | 2 | 4 | 32 | 8192 | 32768 | 65536))

echo "=> Creating 'Moderator' role ..."
if api_call POST "/guilds/$GUILD_ID/roles" \
    "{\"name\":\"Moderator\",\"color\":3447003,\"permissions\":\"$MOD_PERMS\",\"position\":1}" \
    "$ALICE_TOKEN"; then
    ROLE_ID=$(json_field '.id')
    echo "   Created Moderator role (id=$ROLE_ID)"
else
    echo "   Role creation failed, checking existing roles ..."
    api_call GET "/guilds/$GUILD_ID/roles" "" "$ALICE_TOKEN" || die "Cannot list roles"
    ROLE_ID=$(echo "$BODY" | jq -r '.[] | select(.name == "Moderator") | .id' | head -1)
    if [[ -z "$ROLE_ID" || "$ROLE_ID" == "null" ]]; then
        die "Failed to create or find Moderator role."
    fi
    echo "   Found existing Moderator role (id=$ROLE_ID)"
fi

echo "=> Assigning Moderator role to alice ..."
if api_call PUT "/guilds/$GUILD_ID/members/$ALICE_ID/roles/$ROLE_ID" '' "$ALICE_TOKEN"; then
    echo "   Alice now has the Moderator role."
else
    echo "   Role assignment may already exist (skipping)."
fi

# ---------------------------------------------------------------------------
# Step 6: Send messages in #general
# ---------------------------------------------------------------------------

send_msg() {
    local channel_id="$1" token="$2" content="$3"
    api_call POST "/channels/$channel_id/messages" \
        "{\"content\":\"$content\"}" "$token" || true
}

echo "=> Sending sample messages in #general ..."

send_msg "$GENERAL_ID" "$ALICE_TOKEN" "Welcome to Retrocast! This is the demo server."
echo "   [alice] Welcome to Retrocast! This is the demo server."

send_msg "$GENERAL_ID" "$BOB_TOKEN" "Hey alice! Glad to be here."
echo "   [bob] Hey alice! Glad to be here."

send_msg "$GENERAL_ID" "$ALICE_TOKEN" "Feel free to explore the channels. #announcements is for important updates."
echo "   [alice] Feel free to explore the channels. #announcements is for important updates."

send_msg "$GENERAL_ID" "$BOB_TOKEN" "Sounds good. Love the retro vibes!"
echo "   [bob] Sounds good. Love the retro vibes!"

send_msg "$GENERAL_ID" "$ALICE_TOKEN" "Thanks! Check out #random for off-topic chat."
echo "   [alice] Thanks! Check out #random for off-topic chat."

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------

echo ""
echo "=== Demo data seeded successfully ==="
echo ""
echo "Users:"
echo "  alice / password123  (id=$ALICE_ID) — guild owner, Moderator"
echo "  bob   / password123  (id=$BOB_ID)"
echo ""
echo "Guild: Retrocast Demo Server (id=$GUILD_ID)"
echo ""
echo "Channels:"
echo "  #general       (id=$GENERAL_ID)"
echo "  #random        (id=$RANDOM_ID)"
echo "  #announcements (id=$ANNOUNCEMENTS_ID)"
echo ""
echo "Moderator role (id=$ROLE_ID) assigned to alice."
echo ""

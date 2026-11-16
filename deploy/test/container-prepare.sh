#!/bin/sh
# Exercises DockerPrepare.sh, which turns a platform's environment into Dalan settings at container start.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

fail() {
    echo "container-prepare: FAIL: $*" >&2
    exit 1
}

cat > "$work/x-ui" << 'STUB'
#!/bin/sh
echo "$*" >> "$STUB_LOG"
[ "${STUB_FAIL:-}" = "$1" ] && exit 1
exit 0
STUB
chmod +x "$work/x-ui"

# Each case runs in its own shell so exported variables never leak between cases.
prepare() {
    : > "$work/calls"
    env -i PATH="$PATH" STUB_LOG="$work/calls" XUI_BIN="$work/x-ui" XUI_DB_FOLDER="$work/db" "$@" \
        sh -c ". '$here/../../DockerPrepare.sh'; dalan_prepare; echo \"FRONT=\${DALAN_FRONT_PORT:-} PANEL=\${XUI_PORT:-} F2B=\${XUI_ENABLE_FAIL2BAN:-}\""
}

out=$(prepare PORT=8080)
[ "$out" = "FRONT=8080 PANEL=20530 F2B=false" ] || fail "platform PORT: got '$out'"

out=$(prepare PORT=8080 DALAN_FRONT_PORT=9000 XUI_PORT=3000 XUI_ENABLE_FAIL2BAN=true)
[ "$out" = "FRONT=9000 PANEL=3000 F2B=true" ] || fail "explicit values must win over the platform's: got '$out'"

out=$(prepare XUI_ENABLE_FAIL2BAN=true)
[ "$out" = "FRONT= PANEL= F2B=true" ] || fail "without a platform port nothing changes: got '$out'"

prepare DALAN_ADMIN_USER=boss DALAN_ADMIN_PASSWORD=s3cret DALAN_BASE_PATH=/dash/ > /dev/null
[ "$(cat "$work/calls")" = "setting -username boss -password s3cret -webBasePath /dash/" ] || fail "credentials: $(cat "$work/calls")"

prepare DALAN_ADMIN_USER=boss > /dev/null
[ ! -s "$work/calls" ] || fail "a user name without a password changed the panel login: $(cat "$work/calls")"

prepare DALAN_ENROL='dalan://enrol/abc' > /dev/null
[ "$(cat "$work/calls")" = "enrol dalan://enrol/abc" ] || fail "first start must join: $(cat "$work/calls")"
prepare DALAN_ENROL='dalan://enrol/abc' > /dev/null
[ ! -s "$work/calls" ] || fail "a restart joined again with a used link: $(cat "$work/calls")"

rm -rf "$work/db"
out=$(prepare DALAN_ENROL='dalan://enrol/def' STUB_FAIL=enrol) || fail "a failed join stopped the container start"
prepare DALAN_ENROL='dalan://enrol/def' > /dev/null
[ "$(cat "$work/calls")" = "enrol dalan://enrol/def" ] || fail "a failed join must be tried again on the next start: $(cat "$work/calls")"

echo "container-prepare: OK"

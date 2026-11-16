#!/bin/sh
# Turns what a container platform provides into Dalan settings; sourced by DockerEntrypoint.sh.

dalan_prepare() {
    xui="${XUI_BIN:-/app/x-ui}"

    # A platform with one public port (Railway, Orbit) names it PORT: Dalan's front door takes it and
    # the panel moves aside. fail2ban needs iptables, which such platforms do not allow.
    if [ -n "${PORT:-}" ] && [ -z "${DALAN_FRONT_PORT:-}" ]; then
        DALAN_FRONT_PORT="$PORT"
        export DALAN_FRONT_PORT
        if [ -z "${XUI_PORT:-}" ]; then
            XUI_PORT=20530
            export XUI_PORT
        fi
        if [ -z "${XUI_ENABLE_FAIL2BAN:-}" ] || [ "${XUI_ENABLE_FAIL2BAN}" = "true" ]; then
            XUI_ENABLE_FAIL2BAN=false
            export XUI_ENABLE_FAIL2BAN
        fi
    fi

    if [ -n "${DALAN_ADMIN_USER:-}" ] && [ -n "${DALAN_ADMIN_PASSWORD:-}" ]; then
        if [ -n "${DALAN_BASE_PATH:-}" ]; then
            "$xui" setting -username "$DALAN_ADMIN_USER" -password "$DALAN_ADMIN_PASSWORD" -webBasePath "$DALAN_BASE_PATH" > /dev/null
        else
            "$xui" setting -username "$DALAN_ADMIN_USER" -password "$DALAN_ADMIN_PASSWORD" > /dev/null
        fi
    fi

    # An enrolment link works once, so a restart must not use it again; a failed join is tried again.
    if [ -n "${DALAN_ENROL:-}" ]; then
        marker="${XUI_DB_FOLDER:-/etc/x-ui}/enrolled-$(printf '%s' "$DALAN_ENROL" | sha256sum | cut -d' ' -f1)"
        if [ ! -e "$marker" ]; then
            mkdir -p "$(dirname "$marker")"
            if "$xui" enrol "$DALAN_ENROL"; then
                : > "$marker"
            else
                echo "Joining the main panel failed; it will be tried again at the next start." >&2
            fi
        fi
    fi
}

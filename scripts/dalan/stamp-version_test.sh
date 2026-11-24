#!/usr/bin/env bash
# Exercises stamp-version.sh: a release tag becomes the version file, anything else is refused.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
dir="$(mktemp -d)"
trap 'rm -rf "${dir}"' EXIT
file="${dir}/dalan_version"

fail() {
    echo "stamp-version_test: FAIL: $*" >&2
    exit 1
}

printf '0.7.0' > "${file}"
bash "${here}/stamp-version.sh" v0.7.3 "${file}"
[[ "$(cat "${file}")" == "0.7.3" ]] || fail "version file = $(cat "${file}"), want 0.7.3 with no newline"
[[ "$(wc -c < "${file}")" -eq 5 ]] || fail "version file must hold exactly 0.7.3 and no newline"

for bad in "" main v0.7 v0.7.3-rc1 0.7.3 v1.2.x; do
    printf '0.7.3' > "${file}"
    if bash "${here}/stamp-version.sh" "${bad}" "${file}" 2>/dev/null; then
        fail "accepted the ref '${bad}'"
    fi
    [[ "$(cat "${file}")" == "0.7.3" ]] || fail "'${bad}' changed the version file"
done
echo "stamp-version_test: OK"

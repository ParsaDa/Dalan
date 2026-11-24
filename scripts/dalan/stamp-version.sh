#!/usr/bin/env bash
# Writes a release tag (vX.Y.Z) into the version file the panel's update check reads.
set -euo pipefail

tag="${1:-}"
file="${2:?usage: stamp-version.sh vX.Y.Z <version file>}"

if [[ ! "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "stamp-version: '${tag}' is not a release tag like v1.2.3" >&2
    exit 1
fi
printf '%s' "${tag#v}" > "${file}"

#!/usr/bin/env bash
# Fails when an upstream release or image URL survives in scripts or workflows.
set -euo pipefail

root="${1:-$(git rev-parse --show-toplevel)}"
pattern='(raw\.githubusercontent\.com|api\.github\.com/repos|github\.com|ghcr\.io)/mhsanaei/3x-ui|hsanaeii/3x-ui|DOCKER_HUB_'

files=()
while IFS= read -r file; do
    files+=("${file}")
done < <(
    find "${root}" -maxdepth 1 -type f -name '*.sh'
    if [[ -d "${root}/.github/workflows" ]]; then
        find "${root}/.github/workflows" -maxdepth 1 -type f -name '*.yml'
    fi
)

[[ ${#files[@]} -gt 0 ]] || exit 0

offenders="$(grep -HnEi -- "${pattern}" "${files[@]}" | grep -vF 'github.com/mhsanaei/3x-ui/v3' || true)"
if [[ -n "${offenders}" ]]; then
    echo "check-brand: upstream URLs found; run scripts/dalan/rebrand.sh" >&2
    echo "${offenders}" >&2
    exit 1
fi

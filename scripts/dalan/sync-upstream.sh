#!/usr/bin/env bash
# Merges upstream 3x-ui into a new branch and re-applies Dalan branding.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
upstream_url="https://github.com/MHSanaei/3x-ui.git"
upstream_ref="${1:-main}"

if [[ -n "$(git status --porcelain)" ]]; then
    echo "sync-upstream: commit or stash your changes first" >&2
    exit 1
fi

git remote get-url upstream > /dev/null 2>&1 || git remote add upstream "${upstream_url}"
depth=()
if [[ "$(git rev-parse --is-shallow-repository)" == "true" ]]; then
    # A shallow clone has no common ancestor to merge on until both sides are deepened.
    depth=(--depth=300)
    git fetch "${depth[@]}" origin "$(git rev-parse --abbrev-ref HEAD)"
fi
git fetch "${depth[@]}" upstream "${upstream_ref}"
upstream_head="$(git rev-parse FETCH_HEAD)"

if git merge-base --is-ancestor "${upstream_head}" HEAD; then
    echo "sync-upstream: already up to date with upstream ${upstream_ref}"
    exit 0
fi

git checkout -b "sync/upstream-$(date +%Y%m%d)"
git merge --no-commit --no-ff "${upstream_head}" || true

# Branded files are regenerated from upstream's side, so a conflict in them is never resolved by hand.
while IFS= read -r file; do
    case "${file}" in
        install.sh | update.sh | x-ui.sh | .github/workflows/docker.yml)
            git checkout --theirs -- "${file}"
            git add -- "${file}"
            ;;
    esac
done < <(git diff --name-only --diff-filter=U)

bash scripts/dalan/rebrand.sh
git add -- ./*.sh .github/workflows

remaining="$(git diff --name-only --diff-filter=U)"
if [[ -n "${remaining}" ]]; then
    echo "sync-upstream: resolve these conflicts, then run 'make verify' and commit:" >&2
    echo "${remaining}" >&2
    exit 1
fi

bash scripts/dalan/check-brand.sh
echo "sync-upstream: merged cleanly; run 'make verify', then 'git commit'"

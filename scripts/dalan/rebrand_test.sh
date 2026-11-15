#!/usr/bin/env bash
# Exercises rebrand.sh and check-brand.sh against a throwaway tree.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(mktemp -d)"
trap 'rm -rf "${root}"' EXIT

fail() {
    echo "rebrand_test: FAIL: $*" >&2
    exit 1
}

has() { grep -qF -- "$2" "$1" || fail "$1 is missing: $2"; }
lacks() { ! grep -qF -- "$2" "$1" || fail "$1 still has: $2"; }
count() { grep -cF -- "$2" "$1" || true; }

mkdir -p "${root}/.github/workflows"
cat > "${root}/a.sh" <<'TREE'
curl -o p.tgz https://github.com/MHSanaei/3x-ui/releases/download/v1/x-ui-linux-amd64.tar.gz
bash <(curl -Ls https://raw.githubusercontent.com/mhsanaei/3x-ui/main/install.sh)
curl -Ls https://api.github.com/repos/MHSanaei/3x-ui/releases/latest
curl -sfL https://github.com/mhsanaei/mtg-multi/releases/latest
db=/etc/x-ui/x-ui.db
    chmod +x /usr/bin/x-ui
    rm ${xui_folder}/ -rf
echo https://raw.githubusercontent.com/mhsanaei/3x-ui/master/install.sh
TREE
cp "${root}/a.sh" "${root}/x-ui.sh"
cat > "${root}/.github/workflows/r.yml" <<'TREE'
images: ghcr.io/mhsanaei/3x-ui
ldflags: -X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc
TREE

out="$(bash "${here}/check-brand.sh" "${root}" 2>&1)" && fail "check passed on an unbranded tree"
[[ "${out}" == *"a.sh"* && "${out}" == *"r.yml"* ]] || fail "check did not name both files: ${out}"

bash "${here}/rebrand.sh" "${root}"

has "${root}/a.sh" "https://github.com/ParsaDa/dalan/releases/download/v1/x-ui-linux-amd64.tar.gz"
has "${root}/a.sh" "https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh"
has "${root}/a.sh" "https://api.github.com/repos/ParsaDa/dalan/releases/latest"
has "${root}/a.sh" "https://github.com/mhsanaei/mtg-multi/releases/latest"
has "${root}/a.sh" "db=/etc/x-ui/x-ui.db"
has "${root}/a.sh" "    ln -sf /usr/bin/x-ui /usr/bin/dalan"
has "${root}/x-ui.sh" "    rm -f /usr/bin/dalan /usr/bin/x-ui"
lacks "${root}/a.sh" "rm -f /usr/bin/dalan"
has "${root}/a.sh" "echo https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh"
has "${root}/.github/workflows/r.yml" "images: ghcr.io/parsada/dalan"
has "${root}/.github/workflows/r.yml" "-X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc"
lacks "${root}/a.sh" "MHSanaei/3x-ui"

bash "${here}/check-brand.sh" "${root}" || fail "check failed on a rebranded tree"

before="$(cd "${root}" && find . -type f -exec sha256sum {} + | sort)"
bash "${here}/rebrand.sh" "${root}"
after="$(cd "${root}" && find . -type f -exec sha256sum {} + | sort)"
[[ "${before}" == "${after}" ]] || fail "second rebrand run changed files"
[[ "$(count "${root}/x-ui.sh" "/usr/bin/dalan")" == "2" ]] || fail "alias lines were duplicated"

echo 'curl https://raw.githubusercontent.com/MHSanaei/3x-ui/main/x.sh' > "${root}/new.sh"
out="$(bash "${here}/check-brand.sh" "${root}" 2>&1)" && fail "check passed with a new upstream URL"
[[ "${out}" == *"new.sh"* ]] || fail "check did not name new.sh: ${out}"

echo "rebrand_test: OK"

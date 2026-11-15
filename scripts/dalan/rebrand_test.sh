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
cat >> "${root}/x-ui.sh" <<'TREE'
        "restart-xray")
            check_install 0 && restart_xray 0
            ;;
TREE
cat > "${root}/.github/workflows/r.yml" <<'TREE'
images: ghcr.io/mhsanaei/3x-ui
ldflags: -X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc
          images: |
            hsanaeii/3x-ui
            ghcr.io/mhsanaei/3x-ui

      - name: Login to Docker Hub
        uses: docker/login-action@v4
        with:
          username: ${{ secrets.DOCKER_HUB_USERNAME }}
          password: ${{ secrets.DOCKER_HUB_TOKEN }}

      - name: Login to GHCR
        uses: docker/login-action@v4
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
has "${root}/x-ui.sh" '            check_install 0 && "${xui_folder}/x-ui" enrol "$2" && restart_xray 0'
[[ "$(grep -nF -- '"enrol")' "${root}/x-ui.sh" | cut -d: -f1)" -lt "$(grep -nF -- '"restart-xray")' "${root}/x-ui.sh" | cut -d: -f1)" ]] \
    || fail "the enrol command is not a case of its own ahead of restart-xray"
lacks "${root}/a.sh" '"enrol")'
has "${root}/a.sh" "echo https://raw.githubusercontent.com/ParsaDa/dalan/main/install.sh"
has "${root}/.github/workflows/r.yml" "images: ghcr.io/parsada/dalan"
has "${root}/.github/workflows/r.yml" "-X github.com/mhsanaei/3x-ui/v3/internal/config.buildCommit=abc"
lacks "${root}/.github/workflows/r.yml" "hsanaeii/3x-ui"
lacks "${root}/.github/workflows/r.yml" "Login to Docker Hub"
lacks "${root}/.github/workflows/r.yml" "DOCKER_HUB_TOKEN"
has "${root}/.github/workflows/r.yml" "      - name: Login to GHCR"
has "${root}/.github/workflows/r.yml" "        uses: docker/login-action@v4"
lacks "${root}/a.sh" "MHSanaei/3x-ui"

bash "${here}/check-brand.sh" "${root}" || fail "check failed on a rebranded tree"

before="$(cd "${root}" && find . -type f -exec sha256sum {} + | sort)"
bash "${here}/rebrand.sh" "${root}"
after="$(cd "${root}" && find . -type f -exec sha256sum {} + | sort)"
[[ "${before}" == "${after}" ]] || fail "second rebrand run changed files"
[[ "$(count "${root}/x-ui.sh" "/usr/bin/dalan")" == "2" ]] || fail "alias lines were duplicated"
[[ "$(count "${root}/x-ui.sh" '"enrol")')" == "1" ]] || fail "the enrol command was duplicated"

echo 'curl https://raw.githubusercontent.com/MHSanaei/3x-ui/main/x.sh' > "${root}/new.sh"
out="$(bash "${here}/check-brand.sh" "${root}" 2>&1)" && fail "check passed with a new upstream URL"
[[ "${out}" == *"new.sh"* ]] || fail "check did not name new.sh: ${out}"

printf 'images: hsanaeii/3x-ui\n' > "${root}/.github/workflows/hub.yml"
out="$(bash "${here}/check-brand.sh" "${root}" 2>&1)" && fail "check passed with the upstream Docker Hub image"
[[ "${out}" == *"hub.yml"* ]] || fail "check did not name hub.yml: ${out}"

echo "rebrand_test: OK"

#!/usr/bin/env bash
# Re-applies Dalan branding to upstream scripts and workflows; safe to re-run.
set -euo pipefail

root="${1:-$(git rev-parse --show-toplevel)}"
slug="ParsaDa/dalan"
image="ghcr.io/parsada/dalan"
alias_marker="/usr/bin/dalan"

targets() {
    find "${root}" -maxdepth 1 -type f -name '*.sh'
    if [[ -d "${root}/.github/workflows" ]]; then
        find "${root}/.github/workflows" -maxdepth 1 -type f -name '*.yml'
    fi
}

rebrand_urls() {
    # The Go module path ends in /v3 and must survive, so it is parked first.
    sed -E -i \
        -e 's#github\.com/mhsanaei/3x-ui/v3#@@DALAN_GO_MODULE@@#g' \
        -e "s#(raw\.githubusercontent\.com|api\.github\.com/repos|github\.com)/mhsanaei/3x-ui#\1/${slug}#Ig" \
        -e "s#ghcr\.io/mhsanaei/3x-ui#${image}#Ig" \
        -e "s#(raw\.githubusercontent\.com/${slug})/master/#\1/main/#g" \
        -e 's#@@DALAN_GO_MODULE@@#github.com/mhsanaei/3x-ui/v3#g' \
        "$1"
}

add_command_alias() {
    grep -qF -- "${alias_marker}" "$1" && return 0
    sed -E -i \
        -e 's#^([[:space:]]*)chmod \+x /usr/bin/x-ui([[:space:]].*)?$#&\n\1ln -sf /usr/bin/x-ui /usr/bin/dalan#' \
        "$1"
    # Only the management script uninstalls; install and update wipe the same folder to reinstall.
    [[ "$(basename "$1")" == "x-ui.sh" ]] || return 0
    sed -E -i \
        -e 's#^([[:space:]]*)rm \$\{xui_folder\}/ -rf$#&\n\1rm -f /usr/bin/dalan /usr/bin/x-ui#' \
        "$1"
}

add_enrol_command() {
    # "dalan enrol <link>" joins a main panel, then Xray restarts to open the management tunnel.
    [[ "$(basename "$1")" == "x-ui.sh" ]] || return 0
    grep -qF -- '"enrol")' "$1" && return 0
    sed -E -i \
        -e 's#^([[:space:]]*)"restart-xray"\)$#\1"enrol")\n\1    check_install 0 \&\& "${xui_folder}/x-ui" enrol "$2" \&\& restart_xray 0\n\1    ;;\n&#' \
        "$1"
}

add_enrol_on_install() {
    # XUI_ENROL=<link> makes a fresh install join its main panel in the same command; failing never fails the install.
    [[ "$(basename "$1")" == "install.sh" ]] || return 0
    grep -qF -- 'XUI_ENROL' "$1" && return 0
    perl -0pi -e 's#^([ \t]*)(echo -e "\$\{green\}x-ui \$\{tag_version\}\$\{plain\} installation finished)#$1if [[ -n "\${XUI_ENROL:-}" ]]; then\n$1    echo -e "\${green}Joining the main panel...\${plain}"\n$1    /usr/bin/x-ui enrol "\${XUI_ENROL}" || echo -e "\${red}Joining failed. Run again later: x-ui enrol '"'"'<link>'"'"'\${plain}"\n$1fi\n\n$1$2#m' "$1"
}

drop_upstream_docker_hub() {
    # Upstream also publishes to its author's Docker Hub account; Dalan images go to GHCR only.
    perl -0pi -e 's/^[ \t]*hsanaeii\/3x-ui[ \t]*\n//mg; s/^([ \t]*)- name: Login to Docker Hub\n(?:\1[ \t]+.*\n)*\n?//mg' "$1"
}

while IFS= read -r file; do
    rebrand_urls "${file}"
    case "${file}" in
        *.sh)
            add_command_alias "${file}"
            add_enrol_command "${file}"
            add_enrol_on_install "${file}"
            ;;
        *.yml) drop_upstream_docker_hub "${file}" ;;
    esac
done < <(targets)

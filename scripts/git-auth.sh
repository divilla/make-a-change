#!/usr/bin/env bash
set -euo pipefail

key=${MCH_GIT_SSH_KEY:-$HOME/.ssh/divilla-github/id_ed25519}

if [[ $# -eq 1 && $1 == --check ]]; then
    # Verify that this agent can sign with the configured key, not just any key.
    if ssh-add -T "$key.pub" >/dev/null 2>&1; then
        exit 0
    fi
    script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
    printf -v auth_command 'bash %q' "$script_dir/git-auth.sh"
    printf "error: Git authentication is not ready. Run '%s' in this terminal to load your SSH key, then retry this command.\n" \
        "$auth_command" >&2
    exit 1
fi

if [[ $# -ne 0 ]]; then
    echo 'usage: scripts/git-auth.sh [--check]' >&2
    exit 1
fi

if ! ssh-add "$key"; then
    echo 'error: Unable to load the Git SSH key. Make sure an SSH agent is running in this terminal and the key is available, then retry.' >&2
    exit 1
fi

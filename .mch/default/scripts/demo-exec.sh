#!/usr/bin/env bash
set -euo pipefail

stage="${1:-}"
if [[ -z "${stage}" ]]; then
  printf '%s\n' 'stage is required' >&2
  exit 2
fi

printf 'mch demo exec: %s (%s)\n' "${stage}" "${MCH_STAGE:-unset}"

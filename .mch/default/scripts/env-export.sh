#!/usr/bin/env bash

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    printf '%s\n' 'error: this script must be sourced: source scripts/env-export.sh' >&2
    exit 1
fi

export MCH_DEFAULT_DIR=".mch/default"
export MCH_TEMP_DIR=".mch/tmp"
export MCH_REF_UUID="11111111-1111-1111-1111-111111111111"
export MCH_STAGE="artifact"

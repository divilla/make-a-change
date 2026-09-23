#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  printf '%s\n' 'usage: codex-exec-new-session.sh <prompt-path>' >&2
  exit 2
fi

prompt_path="${1}"

if [[ -z "${MCH_DEFAULT_DIR:-}" ]]; then
  printf '%s\n' 'missing MCH_DEFAULT_DIR' >&2
  exit 1
fi

if [[ "${MCH_DEFAULT_DIR}" == /* || "${MCH_DEFAULT_DIR}" == *".."* ]]; then
  printf 'invalid MCH_DEFAULT_DIR: %s\n' "${MCH_DEFAULT_DIR}" >&2
  exit 1
fi

if [[ -z "${MCH_TEMP_UUID:-}" ]]; then
  printf '%s\n' 'missing MCH_TEMP_UUID' >&2
  exit 1
fi

if [[ ! "${MCH_TEMP_UUID}" =~ ^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$ ]]; then
  printf 'invalid MCH_TEMP_UUID: %s\n' "${MCH_TEMP_UUID}" >&2
  exit 1
fi

repo="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "${repo}" ]]; then
  printf '%s\n' 'cannot extract repo root' >&2
  exit 1
fi

if [[ "${prompt_path}" != /* ]]; then
  prompt_path="${PWD}/${prompt_path}"
fi

if [[ ! -f "${prompt_path}" ]]; then
  printf 'missing prompt file: %s\n' "${prompt_path}" >&2
  exit 1
fi

temp_dir="${repo}/.mch/tmp/${MCH_TEMP_UUID}"
mkdir -p "${temp_dir}"

prompt="$(
  sed \
    -e "s|/tmp-dir/|${temp_dir}/|g" \
    -e "s|/def-dir/|${MCH_DEFAULT_DIR%/}/|g" \
    "${prompt_path}"
)"

rm -f "${temp_dir}/agent_output.md" "${temp_dir}/events.jsonl" "${temp_dir}/error.log" "${temp_dir}/session_id"

codex exec -C "${repo}" --json -o "${temp_dir}/agent_output.md" "${prompt}" > "${temp_dir}/events.jsonl" 2> "${temp_dir}/error.log"

if [[ ! -s "${temp_dir}/session_id" ]]; then
  session_id="$(jq -rsr 'map(select(.type=="thread.started") | (.thread_id // .session_id // .session.id // .id // empty)) | first // empty' "${temp_dir}/events.jsonl")"
  if [[ -z "${session_id}" ]]; then
    printf '%s\n' 'missing session_id' >&2
    exit 1
  fi
  printf '%s\n' "${session_id} started"
  printf '%s\n' "${session_id}" > "${temp_dir}/session_id"
fi

if [[ ! -s "${temp_dir}/agent_output.md" ]]; then
  printf '%s\n' 'missing agent output' >&2
  exit 1
fi

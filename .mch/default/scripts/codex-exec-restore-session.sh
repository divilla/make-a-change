#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  printf '%s\n' 'usage: codex-exec-restore-session.sh <prompt-path>' >&2
  exit 2
fi

prompt_path="${1}"

if [[ -z "${MCH_TEMP_DIR:-}" ]]; then
  printf '%s\n' 'missing MCH_TEMP_DIR' >&2
  exit 1
fi

if [[ "${MCH_TEMP_DIR}" == /* || "${MCH_TEMP_DIR}" == *".."* ]]; then
  printf 'invalid MCH_TEMP_DIR: %s\n' "${MCH_TEMP_DIR}" >&2
  exit 1
fi

if [[ -z "${MCH_REF_UUID:-}" ]]; then
  printf '%s\n' 'missing MCH_REF_UUID' >&2
  exit 1
fi

if [[ ! "${MCH_REF_UUID}" =~ ^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$ ]]; then
  printf 'invalid MCH_REF_UUID: %s\n' "${MCH_REF_UUID}" >&2
  exit 1
fi

if [[ -z "${MCH_STAGE:-}" ]]; then
  printf '%s\n' 'missing MCH_STAGE' >&2
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

flow_temp_root="${repo}/${MCH_TEMP_DIR}"
if [[ ! -d "${flow_temp_root}" ]]; then
  printf 'missing Flow temp root: %s\n' "${MCH_TEMP_DIR}" >&2
  exit 1
fi

temp_dir="${flow_temp_root}/${MCH_REF_UUID}/${MCH_STAGE}"

for resource in input.md output.md; do
  if [[ ! -f "${temp_dir}/${resource}" ]]; then
    printf 'missing artifact resource: %s/%s/%s/%s\n' "${MCH_TEMP_DIR}" "${MCH_REF_UUID}" "${MCH_STAGE}" "${resource}" >&2
    exit 1
  fi
done

prompt="$(sed "s|/tmp-dir/|${temp_dir}/|g" "${prompt_path}")"

rm -f "${temp_dir}/agent-output.md" "${temp_dir}/events.jsonl" "${temp_dir}/error.log"

session_id=""
if [[ -f "${temp_dir}/session-id" ]]; then
  session_id="$(grep -Eom1 '[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}' "${temp_dir}/session-id" || true)"
fi

if [[ -n "${session_id}" ]]; then
  codex_sessions_dir="${CODEX_HOME:-${HOME}/.codex}/sessions"
  if [[ ! -d "${codex_sessions_dir}" ]] || ! find "${codex_sessions_dir}" -type f -name "*${session_id}.jsonl" | grep -q .; then
    session_id=""
  fi
fi

if [[ -n "${session_id}" ]]; then
  printf '%s\n' "${session_id} restored"
  codex exec -C "${repo}" --json -o "${temp_dir}/agent-output.md" resume "${session_id}" "${prompt}" > "${temp_dir}/events.jsonl" 2> "${temp_dir}/error.log"
else
  rm -f "${temp_dir}/session-id"
  codex exec -C "${repo}" --json -o "${temp_dir}/agent-output.md" "${prompt}" > "${temp_dir}/events.jsonl" 2> "${temp_dir}/error.log"
fi

if [[ ! -s "${temp_dir}/session-id" ]]; then
  session_id="$(jq -rsr 'map(select(.type=="thread.started") | (.thread_id // .session_id // .session.id // .id // empty)) | first // empty' "${temp_dir}/events.jsonl")"
  if [[ -z "${session_id}" ]]; then
    printf '%s\n' 'missing session-id' >&2
    exit 1
  fi
  printf '%s\n' "${session_id}" > "${temp_dir}/session-id"
fi

if [[ ! -f "${temp_dir}/agent-output.md" ]]; then
  printf '%s\n' 'missing agent output' >&2
  exit 1
fi

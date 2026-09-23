#!/usr/bin/env bash
set -euo pipefail

prompt_path="${1:-}"

if [[ $# -gt 1 ]]; then
  printf '%s\n' 'usage: codex-no-session.sh [prompt-path]' >&2
  exit 2
fi

if [[ -z "${MCH_DEFAULT_DIR:-}" ]]; then
  printf '%s\n' 'missing MCH_DEFAULT_DIR' >&2
  exit 1
fi

if [[ "${MCH_DEFAULT_DIR}" == /* || "${MCH_DEFAULT_DIR}" == *".."* ]]; then
  printf 'invalid MCH_DEFAULT_DIR: %s\n' "${MCH_DEFAULT_DIR}" >&2
  exit 1
fi

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

temp_dir="${repo}/${MCH_TEMP_DIR}/${MCH_REF_UUID}/${MCH_STAGE}"
for resource in input.md output.md; do
  if [[ ! -f "${temp_dir}/${resource}" ]]; then
    printf 'missing artifact resource: %s/%s/%s/%s\n' "${MCH_TEMP_DIR}" "${MCH_REF_UUID}" "${MCH_STAGE}" "${resource}" >&2
    exit 1
  fi
done

if [[ -z "${prompt_path}" ]]; then
  codex -C "${repo}"
  exit 0
fi

if [[ "${prompt_path}" != /* ]]; then
  prompt_path="${PWD}/${prompt_path}"
fi

if [[ ! -f "${prompt_path}" ]]; then
  printf 'missing prompt file: %s\n' "${prompt_path}" >&2
  exit 1
fi

prompt="$(
  sed \
    -e "s|/stg-tmp-dir/|${temp_dir}/|g" \
    -e "s|/def-dir/|${MCH_DEFAULT_DIR%/}/|g" \
    "${prompt_path}"
)"

codex -C "${repo}" "${prompt}"
exit 0

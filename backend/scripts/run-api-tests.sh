#!/usr/bin/env bash
# Retained Go HTTP scenarios use the same private owned lifecycle, never APIHydra coverage.
set -euo pipefail
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."
exec python3 -B scripts/api_coverage.py --legacy

#!/usr/bin/env bash
# conform.sh — validation conformance test: goldmark-lint vs markdownlint-cli2
#
# Runs both linters against the rust-lang/rfcs corpus at a fixed commit and
# reports per-rule deltas and a summary of discrepancies between their outputs.
#
# Requirements:
#   - git, go, jq
#   - markdownlint-cli2 (npm install -g markdownlint-cli2)
#
# Usage:
#   ./bench/conform.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Fixed commit in rust-lang/rfcs used as the conformance corpus.
RFCS_REPO="https://github.com/rust-lang/rfcs"
RFCS_COMMIT="c143e315774f746d667c5eecd95a8ed999e8a729"
RFCS_DIR="${SCRIPT_DIR}/rfcs"

GOLDMARK_BIN="${SCRIPT_DIR}/goldmark-lint"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info() { echo "==> $*"; }

require_cmd() {
  if ! command -v "$1" &>/dev/null; then
    echo "Error: required command '$1' not found on PATH." >&2
    exit 1
  fi
}

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------
require_cmd git
require_cmd go
require_cmd jq
require_cmd markdownlint-cli2

# ---------------------------------------------------------------------------
# Clone / update the rfcs corpus at the fixed commit
# ---------------------------------------------------------------------------
if [[ ! -d "${RFCS_DIR}/.git" ]]; then
  info "Cloning rust-lang/rfcs corpus (shallow)…"
  git clone -q --filter=blob:none --no-checkout "${RFCS_REPO}" "${RFCS_DIR}"
  git -c advice.detachedHead=false -C "${RFCS_DIR}" checkout -q "${RFCS_COMMIT}"
else
  current=$(git -C "${RFCS_DIR}" rev-parse HEAD)
  if [[ "${current}" != "${RFCS_COMMIT}" ]]; then
    info "Updating rfcs corpus to ${RFCS_COMMIT}…"
    git -C "${RFCS_DIR}" fetch -q origin "${RFCS_COMMIT}"
    git -c advice.detachedHead=false -C "${RFCS_DIR}" checkout -q "${RFCS_COMMIT}"
  else
    info "rfcs corpus already at ${RFCS_COMMIT}."
  fi
fi

MD_COUNT=$(find "${RFCS_DIR}" -name '*.md' | wc -l | tr -d ' ')
info "Corpus: ${MD_COUNT} Markdown files in ${RFCS_DIR}"

# ---------------------------------------------------------------------------
# Build goldmark-lint
# ---------------------------------------------------------------------------
info "Building goldmark-lint…"
go build -o "${GOLDMARK_BIN}" "${REPO_ROOT}/cmd/goldmark-lint"
info "Built: ${GOLDMARK_BIN}"

# ---------------------------------------------------------------------------
# Run both linters and capture output
# ---------------------------------------------------------------------------
GOLDMARK_OUT=$(mktemp)
MDLINT_OUT=$(mktemp)
trap 'rm -f "${GOLDMARK_OUT}" "${MDLINT_OUT}"' EXIT

cd "${RFCS_DIR}"

info "Running goldmark-lint…"
# goldmark-lint writes JSON violations to stdout; exit code 1 when violations found.
"${GOLDMARK_BIN}" --no-cache --output-format json '**/*.md' >"${GOLDMARK_OUT}" 2>/dev/null || true

info "Running markdownlint-cli2…"
# markdownlint-cli2 writes violation lines to stderr; capture them for parsing.
markdownlint-cli2 '**/*.md' 2>"${MDLINT_OUT}" >/dev/null || true

# ---------------------------------------------------------------------------
# Extract per-rule violation counts.
#
# goldmark-lint JSON format:
#   A flat array of objects; each has "ruleNames": ["MDxxx", ...].
#   Primary rule ID is ruleNames[0].
#
# markdownlint-cli2 text format (default stderr output):
#   One violation per line: <file>:<line> error MDxxx/<alias> <description>
#   Primary rule ID is the MDxxx token before the slash.
# ---------------------------------------------------------------------------

# extract_gm_rules emits one rule ID per line from goldmark-lint JSON output.
extract_gm_rules() {
  jq -r '.[].ruleNames[0]' "$1"
}

# extract_ml_rules emits one rule ID per line from markdownlint-cli2 text output.
extract_ml_rules() {
  grep -o 'MD[0-9]*/' "$1" | tr -d '/'
}

declare -A GM_RULE ML_RULE

while read -r count rule; do
  GM_RULE["$rule"]=$count
done < <(extract_gm_rules "${GOLDMARK_OUT}" | sort | uniq -c | awk '{print $1, $2}')

while read -r count rule; do
  ML_RULE["$rule"]=$count
done < <(extract_ml_rules "${MDLINT_OUT}" | sort | uniq -c | awk '{print $1, $2}')

GM_TOTAL=$(extract_gm_rules "${GOLDMARK_OUT}" | wc -l | tr -d ' ')
ML_TOTAL=$(extract_ml_rules "${MDLINT_OUT}" | wc -l | tr -d ' ')

# Collect all rules seen by either tool, sorted alphabetically.
SORTED_RULES=()
mapfile -t SORTED_RULES < <(
  { extract_gm_rules "${GOLDMARK_OUT}"; extract_ml_rules "${MDLINT_OUT}"; } | sort -u
)

# ---------------------------------------------------------------------------
# Print conformance report
# ---------------------------------------------------------------------------
echo ""
echo "=== Conformance Report: goldmark-lint vs markdownlint-cli2 ==="
printf "Corpus : %s (%s files, commit %.10s)\n" "${RFCS_DIR}" "${MD_COUNT}" "${RFCS_COMMIT}"
echo ""
printf "%-10s  %13s  %13s  %10s\n" "RULE" "goldmark-lint" "markdownlint" "delta"
printf "%-10s  %13s  %13s  %10s\n" "----------" "-------------" "-------------" "----------"

for rule in "${SORTED_RULES[@]}"; do
  gm="${GM_RULE[$rule]:-0}"
  ml="${ML_RULE[$rule]:-0}"
  delta=$(( gm - ml ))
  printf "%-10s  %13d  %13d  %+10d\n" "$rule" "$gm" "$ml" "$delta"
done

printf "%-10s  %13s  %13s  %10s\n" "----------" "-------------" "-------------" "----------"
TOTAL_DELTA=$(( GM_TOTAL - ML_TOTAL ))
printf "%-10s  %13d  %13d  %+10d\n" "TOTAL" "$GM_TOTAL" "$ML_TOTAL" "$TOTAL_DELTA"
echo ""

# Summary: rules flagged exclusively by one tool (discrepancies).
ONLY_GM=()
ONLY_ML=()
for rule in "${SORTED_RULES[@]}"; do
  gm="${GM_RULE[$rule]:-0}"
  ml="${ML_RULE[$rule]:-0}"
  if [[ "$ml" -eq 0 && "$gm" -gt 0 ]]; then
    ONLY_GM+=("${rule} (${gm})")
  elif [[ "$gm" -eq 0 && "$ml" -gt 0 ]]; then
    ONLY_ML+=("${rule} (${ml})")
  fi
done

if [[ ${#ONLY_GM[@]} -gt 0 ]]; then
  echo "Flagged only by goldmark-lint:"
  printf '  %s\n' "${ONLY_GM[@]}"
  echo ""
fi
if [[ ${#ONLY_ML[@]} -gt 0 ]]; then
  echo "Flagged only by markdownlint-cli2:"
  printf '  %s\n' "${ONLY_ML[@]}"
  echo ""
fi

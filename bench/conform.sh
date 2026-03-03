#!/usr/bin/env bash
# conform.sh — validation conformance test: goldmark-lint vs markdownlint-cli2
#
# Runs both linters against the rust-lang/rfcs and tldr-pages/tldr corpora at
# fixed commits and reports per-rule deltas and a summary of discrepancies
# between their outputs.
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

# Fixed commit in rust-lang/rfcs used as the first conformance corpus.
RFCS_REPO="https://github.com/rust-lang/rfcs"
RFCS_COMMIT="c143e315774f746d667c5eecd95a8ed999e8a729"
RFCS_DIR="${SCRIPT_DIR}/rfcs"

# Fixed commit in tldr-pages/tldr used as the second conformance corpus.
TLDR_REPO="https://github.com/tldr-pages/tldr"
TLDR_COMMIT="05c563d1ecb0fe5c1f0de9d3348baa04f3b8b29d"
TLDR_DIR="${SCRIPT_DIR}/tldr"

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

# ---------------------------------------------------------------------------
# Clone / update the tldr corpus at the fixed commit
# ---------------------------------------------------------------------------
if [[ ! -d "${TLDR_DIR}/.git" ]]; then
  info "Cloning tldr-pages/tldr corpus (shallow)…"
  git clone -q --filter=blob:none --no-checkout "${TLDR_REPO}" "${TLDR_DIR}"
  git -c advice.detachedHead=false -C "${TLDR_DIR}" checkout -q "${TLDR_COMMIT}"
else
  current=$(git -C "${TLDR_DIR}" rev-parse HEAD)
  if [[ "${current}" != "${TLDR_COMMIT}" ]]; then
    info "Updating tldr corpus to ${TLDR_COMMIT}…"
    git -C "${TLDR_DIR}" fetch -q origin "${TLDR_COMMIT}"
    git -c advice.detachedHead=false -C "${TLDR_DIR}" checkout -q "${TLDR_COMMIT}"
  else
    info "tldr corpus already at ${TLDR_COMMIT}."
  fi
fi

RFCS_MD_COUNT=$(find "${RFCS_DIR}" -name '*.md' | wc -l | tr -d ' ')
TLDR_MD_COUNT=$(find "${TLDR_DIR}" -name '*.md' | wc -l | tr -d ' ')
MD_COUNT=$(( RFCS_MD_COUNT + TLDR_MD_COUNT ))
info "Corpus 1: ${RFCS_MD_COUNT} Markdown files in ${RFCS_DIR}"
info "Corpus 2: ${TLDR_MD_COUNT} Markdown files in ${TLDR_DIR}"

# ---------------------------------------------------------------------------
# Build goldmark-lint
# ---------------------------------------------------------------------------
info "Building goldmark-lint…"
go build -o "${GOLDMARK_BIN}" "${REPO_ROOT}/cmd/goldmark-lint"
info "Built: ${GOLDMARK_BIN}"

# ---------------------------------------------------------------------------
# Run both linters against each corpus and capture output
# ---------------------------------------------------------------------------
GOLDMARK_OUT_RFCS=$(mktemp)
GOLDMARK_OUT_TLDR=$(mktemp)
MDLINT_OUT_RFCS=$(mktemp)
MDLINT_OUT_TLDR=$(mktemp)
trap 'rm -f "${GOLDMARK_OUT_RFCS}" "${GOLDMARK_OUT_TLDR}" "${MDLINT_OUT_RFCS}" "${MDLINT_OUT_TLDR}"' EXIT

info "Running goldmark-lint on rfcs…"
# goldmark-lint writes JSON violations to stdout; exit code 1 when violations found.
(cd "${RFCS_DIR}" && "${GOLDMARK_BIN}" --no-cache --output-format json '**/*.md') >"${GOLDMARK_OUT_RFCS}" 2>/dev/null || true

info "Running goldmark-lint on tldr…"
(cd "${TLDR_DIR}" && "${GOLDMARK_BIN}" --no-cache --output-format json '**/*.md') >"${GOLDMARK_OUT_TLDR}" 2>/dev/null || true

info "Running markdownlint-cli2 on rfcs…"
# markdownlint-cli2 writes violation lines to stderr; capture them for parsing.
(cd "${RFCS_DIR}" && markdownlint-cli2 '**/*.md') 2>"${MDLINT_OUT_RFCS}" >/dev/null || true

info "Running markdownlint-cli2 on tldr…"
(cd "${TLDR_DIR}" && markdownlint-cli2 '**/*.md') 2>"${MDLINT_OUT_TLDR}" >/dev/null || true

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
# Accepts one or more file arguments.
extract_gm_rules() {
  jq -r '.[].ruleNames[0]' "$@"
}

# extract_ml_rules emits one rule ID per line from markdownlint-cli2 text output.
# Accepts one or more file arguments; -h suppresses filename prefixes.
extract_ml_rules() {
  grep -oh 'MD[0-9]*/' "$@" | tr -d '/'
}

declare -A GM_RULE ML_RULE

while read -r count rule; do
  GM_RULE["$rule"]=$count
done < <(extract_gm_rules "${GOLDMARK_OUT_RFCS}" "${GOLDMARK_OUT_TLDR}" | sort | uniq -c | awk '{print $1, $2}')

while read -r count rule; do
  ML_RULE["$rule"]=$count
done < <(extract_ml_rules "${MDLINT_OUT_RFCS}" "${MDLINT_OUT_TLDR}" | sort | uniq -c | awk '{print $1, $2}')

GM_TOTAL=$(extract_gm_rules "${GOLDMARK_OUT_RFCS}" "${GOLDMARK_OUT_TLDR}" | wc -l | tr -d ' ')
ML_TOTAL=$(extract_ml_rules "${MDLINT_OUT_RFCS}" "${MDLINT_OUT_TLDR}" | wc -l | tr -d ' ')

# Collect all rules seen by either tool, sorted alphabetically.
SORTED_RULES=()
mapfile -t SORTED_RULES < <(
  { extract_gm_rules "${GOLDMARK_OUT_RFCS}" "${GOLDMARK_OUT_TLDR}"; extract_ml_rules "${MDLINT_OUT_RFCS}" "${MDLINT_OUT_TLDR}"; } | sort -u
)

# ---------------------------------------------------------------------------
# Print conformance report
# ---------------------------------------------------------------------------
echo ""
echo "=== Conformance Report: goldmark-lint vs markdownlint-cli2 ==="
printf "Corpus 1: %s (%s files, commit %.10s)\n" "${RFCS_DIR}" "${RFCS_MD_COUNT}" "${RFCS_COMMIT}"
printf "Corpus 2: %s (%s files, commit %.10s)\n" "${TLDR_DIR}" "${TLDR_MD_COUNT}" "${TLDR_COMMIT}"
printf "Total   : %s Markdown files\n" "${MD_COUNT}"
echo ""
printf "%-10s  %13s  %13s  %10s\n" "RULE" "goldmark-lint" "markdownlint" "delta"
printf "%-10s  %13s  %13s  %10s\n" "----------" "-------------" "-------------" "----------"

TOTAL_ABS_DELTA=0
for rule in "${SORTED_RULES[@]}"; do
  gm="${GM_RULE[$rule]:-0}"
  ml="${ML_RULE[$rule]:-0}"
  delta=$(( gm - ml ))
  abs_delta=${delta#-}
  TOTAL_ABS_DELTA=$(( TOTAL_ABS_DELTA + abs_delta ))
  printf "%-10s  %13d  %13d  %+10d\n" "$rule" "$gm" "$ml" "$delta"
done

printf "%-10s  %13s  %13s  %10s\n" "----------" "-------------" "-------------" "----------"
printf "%-10s  %13d  %13d  %+10d\n" "TOTAL" "$GM_TOTAL" "$ML_TOTAL" "$TOTAL_ABS_DELTA"
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

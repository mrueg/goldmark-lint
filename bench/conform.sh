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

# ---------------------------------------------------------------------------
# Per-location comparison
#
# The table above compares per-rule *counts*, which can hide disagreements: a
# false positive in one file and a missed violation in another cancel out and
# the rule still shows a delta of 0. Every MD005 and MD041 conformance bug
# fixed so far was invisible that way. This section compares the actual
# (file, line, rule) triples instead.
#
# Both tools can report the same rule several times on one line (MD060 reports
# per pipe, for instance), so triples are deduplicated before comparing: this
# measures whether the tools agree that a rule fires on a line, not how many
# times it fires.
# ---------------------------------------------------------------------------

# extract_gm_locations emits "<file>:<line>:<RULE>" for each goldmark-lint
# violation. Accepts a single JSON file.
extract_gm_locations() {
  jq -r '.[] | "\(.fileName):\(.lineNumber):\(.ruleNames[0])"' "$1"
}

# extract_ml_locations emits "<file>:<line>:<RULE>" for each markdownlint-cli2
# violation. Lines come in two shapes,
#
#	path/to/file.md:12 error MD041/first-line-heading ...
#	path/to/file.md:12:1 error MD004/ul-style ...
#
# so the trailing numeric fields are peeled off from the right rather than
# matched positionally; a regex anchored from the left reads the column as the
# line number whenever a column is present.
extract_ml_locations() {
  awk '
    match($0, / (error|warning) MD[0-9]+\//) {
      loc = substr($0, 1, RSTART - 1)
      rest = substr($0, RSTART)
      match(rest, /MD[0-9]+/)
      rule = substr(rest, RSTART, RLENGTH)

      n = split(loc, part, ":")
      if (n < 2) next
      # Drop a trailing column field, then take the line number.
      last = n
      if (part[last] ~ /^[0-9]+$/ && last > 2 && part[last - 1] ~ /^[0-9]+$/) last--
      if (part[last] !~ /^[0-9]+$/) next
      line = part[last]

      file = part[1]
      for (i = 2; i < last; i++) file = file ":" part[i]
      print file ":" line ":" rule
    }
  ' "$1"
}

GM_LOCS=$(mktemp)
ML_LOCS=$(mktemp)
ONLY_GM_LOCS=$(mktemp)
ONLY_ML_LOCS=$(mktemp)
trap 'rm -f "${GOLDMARK_OUT_RFCS}" "${GOLDMARK_OUT_TLDR}" "${MDLINT_OUT_RFCS}" "${MDLINT_OUT_TLDR}" "${GM_LOCS}" "${ML_LOCS}" "${ONLY_GM_LOCS}" "${ONLY_ML_LOCS}"' EXIT

# Prefix each corpus so that identical relative paths in the two corpora cannot
# collide with each other.
{
  extract_gm_locations "${GOLDMARK_OUT_RFCS}" | sed 's|^|rfcs/|'
  extract_gm_locations "${GOLDMARK_OUT_TLDR}" | sed 's|^|tldr/|'
} | LC_ALL=C sort -u >"${GM_LOCS}"
{
  extract_ml_locations "${MDLINT_OUT_RFCS}" | sed 's|^|rfcs/|'
  extract_ml_locations "${MDLINT_OUT_TLDR}" | sed 's|^|tldr/|'
} | LC_ALL=C sort -u >"${ML_LOCS}"

LC_ALL=C comm -23 "${GM_LOCS}" "${ML_LOCS}" >"${ONLY_GM_LOCS}"
LC_ALL=C comm -13 "${GM_LOCS}" "${ML_LOCS}" >"${ONLY_ML_LOCS}"

GM_ONLY_COUNT=$(wc -l <"${ONLY_GM_LOCS}" | tr -d ' ')
ML_ONLY_COUNT=$(wc -l <"${ONLY_ML_LOCS}" | tr -d ' ')
SHARED_COUNT=$(LC_ALL=C comm -12 "${GM_LOCS}" "${ML_LOCS}" | wc -l | tr -d ' ')

echo "=== Per-location agreement ==="
printf "Agreed          : %d\n" "${SHARED_COUNT}"
printf "goldmark-lint only (false positives) : %d\n" "${GM_ONLY_COUNT}"
printf "markdownlint only (missed)           : %d\n" "${ML_ONLY_COUNT}"
echo ""

# print_loc_breakdown summarises a disagreement file by rule and lists a few
# example locations for each, so a failure points straight at a repro.
print_loc_breakdown() {
  local title="$1" file="$2"
  [[ -s "$file" ]] || return 0
  echo "${title}:"
  while read -r rule count; do
    printf '  %-8s %d\n' "$rule" "$count"
    grep ":${rule}\$" "$file" | head -3 | sed 's|^|      |'
    if [[ "$count" -gt 3 ]]; then
      printf '      ... and %d more\n' "$(( count - 3 ))"
    fi
  done < <(sed -E 's|.*:([A-Z]+[0-9]+)$|\1|' "$file" | LC_ALL=C sort | uniq -c | sort -rn | awk '{print $2, $1}')
  echo ""
}

print_loc_breakdown "Reported only by goldmark-lint" "${ONLY_GM_LOCS}"
print_loc_breakdown "Reported only by markdownlint-cli2" "${ONLY_ML_LOCS}"

if [[ "${GM_ONLY_COUNT}" -eq 0 && "${ML_ONLY_COUNT}" -eq 0 ]]; then
  echo "The two tools agree on every violation location."
fi

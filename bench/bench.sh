#!/usr/bin/env bash
# bench.sh — benchmark goldmark-lint vs markdownlint-cli2
#
# Uses rust-lang/rfcs at a fixed commit as the test corpus (580+ Markdown files).
#
# Requirements:
#   - git, go
#   - hyperfine (https://github.com/sharkdp/hyperfine) — recommended
#     Falls back to the shell built-in `time` when hyperfine is not available.
#   - markdownlint-cli2 (npm install -g markdownlint-cli2) — optional;
#     skipped when not found on PATH.
#
# Usage:
#   ./bench/bench.sh [--runs N] [--warmup N] [--no-cache]
#
# Options:
#   --runs N      Number of benchmark runs (default: 10; hyperfine only)
#   --warmup N    Number of warmup runs before timing (default: 3; hyperfine only)
#   --no-cache    Disable goldmark-lint's content cache during the benchmark

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Fixed commit in rust-lang/rfcs used as the benchmark corpus.
RFCS_REPO="https://github.com/rust-lang/rfcs"
RFCS_COMMIT="c143e315774f746d667c5eecd95a8ed999e8a729"
RFCS_DIR="${SCRIPT_DIR}/rfcs"

GOLDMARK_BIN="${REPO_ROOT}/bench/goldmark-lint"

# Defaults for hyperfine options.
RUNS=10
WARMUP=3
NO_CACHE=0

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --runs)    RUNS="$2";   shift 2 ;;
    --warmup)  WARMUP="$2"; shift 2 ;;
    --no-cache) NO_CACHE=1; shift ;;
    *) echo "Unknown argument: $1" >&2; exit 1 ;;
  esac
done

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo "==> $*"; }
warn()  { echo "WARN: $*" >&2; }

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

# ---------------------------------------------------------------------------
# Clone / update the rfcs corpus at the fixed commit
# ---------------------------------------------------------------------------
if [[ ! -d "${RFCS_DIR}/.git" ]]; then
  info "Cloning rust-lang/rfcs corpus (shallow)…"
  git clone --filter=blob:none --no-checkout "${RFCS_REPO}" "${RFCS_DIR}"
  git -C "${RFCS_DIR}" checkout "${RFCS_COMMIT}"
else
  current=$(git -C "${RFCS_DIR}" rev-parse HEAD)
  if [[ "${current}" != "${RFCS_COMMIT}" ]]; then
    info "Updating rfcs corpus to ${RFCS_COMMIT}…"
    git -C "${RFCS_DIR}" fetch origin "${RFCS_COMMIT}"
    git -C "${RFCS_DIR}" checkout "${RFCS_COMMIT}"
  else
    info "rfcs corpus already at ${RFCS_COMMIT}."
  fi
fi

# Count Markdown files so the user can see the scale of the corpus.
MD_COUNT=$(find "${RFCS_DIR}" -name '*.md' | wc -l | tr -d ' ')
info "Corpus: ${MD_COUNT} Markdown files in ${RFCS_DIR}"

# ---------------------------------------------------------------------------
# Build goldmark-lint
# ---------------------------------------------------------------------------
info "Building goldmark-lint…"
go build -o "${GOLDMARK_BIN}" "${REPO_ROOT}/cmd/goldmark-lint"
info "Built: ${GOLDMARK_BIN}"

# ---------------------------------------------------------------------------
# Wrapper functions — invoked directly in the time fallback and referenced
# by name when building hyperfine command strings.
# ---------------------------------------------------------------------------

# run_goldmark lints the corpus with the local goldmark-lint build.
# The glob pattern **/*.md is passed as a literal argument; goldmark-lint
# performs its own expansion via filepath.Glob (and falls back to a walk for
# patterns containing **).
run_goldmark() {
  if [[ "${NO_CACHE}" -eq 1 ]]; then
    "${GOLDMARK_BIN}" --no-cache '**/*.md'
  else
    "${GOLDMARK_BIN}" '**/*.md'
  fi
}

# run_markdownlint lints the corpus with markdownlint-cli2.
run_markdownlint() {
  markdownlint-cli2 '**/*.md'
}

# Command strings used by hyperfine (--shell bash).  The single quotes around
# **/*.md are interpreted by the bash subprocess, preventing shell glob
# expansion so that each tool receives the raw pattern and performs its own
# expansion — matching exactly what a user would type on the command line.
GOLDMARK_EXTRA=""
if [[ "${NO_CACHE}" -eq 1 ]]; then
  GOLDMARK_EXTRA=" --no-cache"
fi
GOLDMARK_CMD="${GOLDMARK_BIN}${GOLDMARK_EXTRA} '**/*.md'"
MARKDOWNLINT_CMD="markdownlint-cli2 '**/*.md'"

HAS_HYPERFINE=0
if command -v hyperfine &>/dev/null; then
  HAS_HYPERFINE=1
fi

HAS_MARKDOWNLINT=0
if command -v markdownlint-cli2 &>/dev/null; then
  HAS_MARKDOWNLINT=1
fi

# ---------------------------------------------------------------------------
# Run benchmarks
# ---------------------------------------------------------------------------
cd "${RFCS_DIR}"

if [[ "${HAS_HYPERFINE}" -eq 1 ]]; then
  info "Using hyperfine (runs=${RUNS}, warmup=${WARMUP})…"

  HYPERFINE_ARGS=(
    --runs "${RUNS}"
    --warmup "${WARMUP}"
    --shell bash
    --command-name "goldmark-lint"
    "${GOLDMARK_CMD}"
  )

  if [[ "${HAS_MARKDOWNLINT}" -eq 1 ]]; then
    HYPERFINE_ARGS+=(
      --command-name "markdownlint-cli2"
      "${MARKDOWNLINT_CMD}"
    )
  else
    warn "markdownlint-cli2 not found on PATH — skipping comparison."
    warn "Install it with: npm install -g markdownlint-cli2"
  fi

  hyperfine "${HYPERFINE_ARGS[@]}"

else
  warn "hyperfine not found — falling back to shell 'time'."
  warn "Install hyperfine for more accurate results: https://github.com/sharkdp/hyperfine"
  echo

  run_timed() {
    local label="$1"
    local cmd="$2"
    echo "--- ${label} ---"
    # Suppress command stdout/stderr; keep time output visible on stderr.
    # The || true prevents set -e from exiting when lint finds violations.
    { time ${cmd} >/dev/null 2>&1 || true; } 2>&1
    echo
  }

  run_timed "goldmark-lint" run_goldmark

  if [[ "${HAS_MARKDOWNLINT}" -eq 1 ]]; then
    run_timed "markdownlint-cli2" run_markdownlint
  else
    warn "markdownlint-cli2 not found on PATH — skipping comparison."
    warn "Install it with: npm install -g markdownlint-cli2"
  fi
fi

#!/usr/bin/env bash
# bench.sh — benchmark goldmark-lint vs markdownlint-cli2
#
# Uses two corpora at fixed commits as the benchmark data sets:
#   - rust-lang/rfcs (580+ Markdown files)
#   - tldr-pages/tldr (2000+ Markdown files)
#   - commonmark/commonmark-spec (one ~9,800-line document)
#
# The first two measure per-file throughput over many small documents. The
# third measures the other dimension entirely: how one large document scales.
# They are reported as separate benchmarks because combining them would hide
# both.
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

# Fixed commit in rust-lang/rfcs used as the first benchmark corpus.
RFCS_REPO="https://github.com/rust-lang/rfcs"
RFCS_COMMIT="c143e315774f746d667c5eecd95a8ed999e8a729"
RFCS_DIR="${SCRIPT_DIR}/rfcs"

# Fixed commit in tldr-pages/tldr used as the second benchmark corpus.
TLDR_REPO="https://github.com/tldr-pages/tldr"
TLDR_COMMIT="05c563d1ecb0fe5c1f0de9d3348baa04f3b8b29d"
TLDR_DIR="${SCRIPT_DIR}/tldr"

# Fixed commit in commonmark/commonmark-spec used as the third benchmark
# corpus: a single ~9,800-line document, benchmarked separately from the
# many-small-files corpora above.
COMMONMARK_REPO="https://github.com/commonmark/commonmark-spec"
COMMONMARK_COMMIT="108bec0c217ed1c611cf666c231fd5e240d475dc"
COMMONMARK_DIR="${SCRIPT_DIR}/commonmark"
COMMONMARK_FILE="spec.txt"

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

# Count Markdown files so the user can see the scale of the corpus.
MD_COUNT=$(find "${RFCS_DIR}" -name '*.md' | wc -l | tr -d ' ')
info "Corpus: ${MD_COUNT} Markdown files in ${RFCS_DIR}"

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

# Count Markdown files in the tldr corpus.
TLDR_MD_COUNT=$(find "${TLDR_DIR}" -name '*.md' | wc -l | tr -d ' ')
info "Corpus: ${TLDR_MD_COUNT} Markdown files in ${TLDR_DIR}"

# ---------------------------------------------------------------------------
# Clone / update the commonmark-spec corpus at the fixed commit
# ---------------------------------------------------------------------------
if [[ ! -d "${COMMONMARK_DIR}/.git" ]]; then
  info "Cloning commonmark/commonmark-spec corpus (shallow)…"
  git clone -q --filter=blob:none --no-checkout "${COMMONMARK_REPO}" "${COMMONMARK_DIR}"
  git -c advice.detachedHead=false -C "${COMMONMARK_DIR}" checkout -q "${COMMONMARK_COMMIT}"
else
  current=$(git -C "${COMMONMARK_DIR}" rev-parse HEAD)
  if [[ "${current}" != "${COMMONMARK_COMMIT}" ]]; then
    info "Updating commonmark-spec corpus to ${COMMONMARK_COMMIT}…"
    git -C "${COMMONMARK_DIR}" fetch -q origin "${COMMONMARK_COMMIT}"
    git -c advice.detachedHead=false -C "${COMMONMARK_DIR}" checkout -q "${COMMONMARK_COMMIT}"
  else
    info "commonmark-spec corpus already at ${COMMONMARK_COMMIT}."
  fi
fi

if [[ ! -f "${COMMONMARK_DIR}/${COMMONMARK_FILE}" ]]; then
  echo "Error: ${COMMONMARK_DIR}/${COMMONMARK_FILE} is missing after checkout." >&2
  exit 1
fi
SPEC_LINES=$(wc -l <"${COMMONMARK_DIR}/${COMMONMARK_FILE}" | tr -d ' ')
info "Corpus: ${COMMONMARK_FILE} (${SPEC_LINES} lines) in ${COMMONMARK_DIR}"

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

# run_goldmark lints both corpora with the local goldmark-lint build in a
# single invocation.  Both glob patterns are passed as literal arguments;
# goldmark-lint performs its own expansion via filepath.Glob (and falls back
# to a walk for patterns containing **).
run_goldmark() {
  if [[ "${NO_CACHE}" -eq 1 ]]; then
    "${GOLDMARK_BIN}" --no-cache 'rfcs/**/*.md' 'tldr/**/*.md'
  else
    "${GOLDMARK_BIN}" 'rfcs/**/*.md' 'tldr/**/*.md'
  fi
}

# run_markdownlint lints both corpora with markdownlint-cli2.
run_markdownlint() {
  markdownlint-cli2 'rfcs/**/*.md' 'tldr/**/*.md'
}

# run_goldmark_spec lints the single large document with the local build.
run_goldmark_spec() {
  if [[ "${NO_CACHE}" -eq 1 ]]; then
    "${GOLDMARK_BIN}" --no-cache "commonmark/${COMMONMARK_FILE}"
  else
    "${GOLDMARK_BIN}" "commonmark/${COMMONMARK_FILE}"
  fi
}

# run_markdownlint_spec lints the single large document with markdownlint-cli2.
run_markdownlint_spec() {
  markdownlint-cli2 "commonmark/${COMMONMARK_FILE}"
}

# Command strings used by hyperfine (--shell bash).  The single quotes around
# the glob patterns are interpreted by the bash subprocess, preventing shell
# glob expansion so that each tool receives the raw patterns and performs its
# own expansion — matching exactly what a user would type on the command line.
GOLDMARK_EXTRA=""
if [[ "${NO_CACHE}" -eq 1 ]]; then
  GOLDMARK_EXTRA=" --no-cache"
fi
GOLDMARK_CMD="${GOLDMARK_BIN}${GOLDMARK_EXTRA} 'rfcs/**/*.md' 'tldr/**/*.md'"
MARKDOWNLINT_CMD="markdownlint-cli2 'rfcs/**/*.md' 'tldr/**/*.md'"

# The spec is a single file rather than a glob, and .txt rather than .md, so
# both tools are handed the path directly.
SPEC_PATH="commonmark/${COMMONMARK_FILE}"
GOLDMARK_SPEC_CMD="${GOLDMARK_BIN}${GOLDMARK_EXTRA} ${SPEC_PATH}"
MARKDOWNLINT_SPEC_CMD="markdownlint-cli2 ${SPEC_PATH}"

HAS_HYPERFINE=0
if command -v hyperfine &>/dev/null; then
  HAS_HYPERFINE=1
fi

HAS_MARKDOWNLINT=0
if command -v markdownlint-cli2 &>/dev/null; then
  HAS_MARKDOWNLINT=1
fi

# ---------------------------------------------------------------------------
# Run benchmarks against both corpora in a single invocation
# ---------------------------------------------------------------------------
cd "${SCRIPT_DIR}"

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

  hyperfine "${HYPERFINE_ARGS[@]}" --ignore-failure

  # Second benchmark: the single large document. Reported separately because
  # it measures how one document scales rather than per-file overhead, and the
  # two numbers move for different reasons.
  echo ""
  info "Single large document (${COMMONMARK_FILE}, ${SPEC_LINES} lines)…"

  SPEC_ARGS=(
    --runs "${RUNS}"
    --warmup "${WARMUP}"
    --shell bash
    --command-name "goldmark-lint"
    "${GOLDMARK_SPEC_CMD}"
  )
  if [[ "${HAS_MARKDOWNLINT}" -eq 1 ]]; then
    SPEC_ARGS+=(
      --command-name "markdownlint-cli2"
      "${MARKDOWNLINT_SPEC_CMD}"
    )
  fi

  hyperfine "${SPEC_ARGS[@]}" --ignore-failure

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
  fi

  info "Single large document (${COMMONMARK_FILE}, ${SPEC_LINES} lines)…"
  run_timed "goldmark-lint (spec)" run_goldmark_spec

  if [[ "${HAS_MARKDOWNLINT}" -eq 1 ]]; then
    run_timed "markdownlint-cli2 (spec)" run_markdownlint_spec
  else
    warn "markdownlint-cli2 not found on PATH — skipping comparison."
    warn "Install it with: npm install -g markdownlint-cli2"
  fi
fi

#!/usr/bin/env bash
#
# Enforce the file-size rule from .kiro/steering/dev-rules.md:
#   "Files: one responsibility per file, max ~300 lines before splitting"
#
# A rule enforced only by review discipline erodes the moment a deadline appears,
# so this makes it mechanical. But 18 files already exceeded the limit when the
# check was introduced, and blocking all work to split them at once would be its
# own kind of recklessness.
#
# So this is a RATCHET, not a cliff:
#   * A file not in the baseline may not exceed MAX_LINES.
#   * A file in the baseline may not grow beyond its recorded size.
#   * Shrinking a baselined file below MAX_LINES and removing its baseline entry
#     is the intended way to pay the debt down.
#
# Net effect: existing debt is frozen and can only shrink; new debt cannot appear.
#
# Usage:
#   scripts/check-file-size.sh            # check (used by CI)
#   scripts/check-file-size.sh --update   # rewrite the baseline from current state
#
# Exit status: 0 when clean, 1 when a violation is found.

set -euo pipefail

MAX_LINES=300

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

baseline_file="scripts/file-size-baseline.txt"

# List Go production files (tests excluded — table-driven test files are
# legitimately long and are not what the rule targets).
#
# Includes untracked-but-not-ignored files (--others --exclude-standard): a
# newly written oversized file must be caught before it is committed, not after.
list_files() {
  git ls-files --cached --others --exclude-standard '*.go' \
    | grep -v '_test\.go$' | sort -u
}

# measure prints "lines path" for every production Go file over MAX_LINES.
measure() {
  local f n
  while IFS= read -r f; do
    [[ -f "$f" ]] || continue
    n=$(wc -l < "$f" | tr -d ' ')
    if (( n > MAX_LINES )); then
      printf '%s %s\n' "$n" "$f"
    fi
  done < <(list_files)
}

if [[ "${1:-}" == "--update" ]]; then
  {
    echo "# Files exceeding ${MAX_LINES} lines at the time this baseline was taken."
    echo "# Format: <lines> <path>. These may shrink but never grow."
    echo "# Regenerate with: scripts/check-file-size.sh --update"
    measure
  } > "${baseline_file}"
  echo "Baseline written to ${baseline_file} ($(measure | wc -l | tr -d ' ') entries)"
  exit 0
fi

if [[ ! -f "${baseline_file}" ]]; then
  echo "error: ${baseline_file} not found. Generate it with: scripts/check-file-size.sh --update" >&2
  exit 1
fi

# Baseline lookup uses grep rather than an associative array so the script runs
# on bash 3.2, which is still /bin/bash on macOS runners.
baseline_for() {
  awk -v want="$1" '$1 !~ /^#/ && $2 == want { print $1; exit }' "${baseline_file}"
}

violations=0

# 1. New or grown violations.
while read -r lines path; do
  base=$(baseline_for "${path}")
  if [[ -n "${base}" ]]; then
    if (( lines > base )); then
      printf '  GREW    %-52s %s -> %s lines (baseline exceeded)\n' \
        "${path}" "${base}" "${lines}"
      violations=$((violations + 1))
    fi
  else
    printf '  NEW     %-52s %s lines (limit %s)\n' "${path}" "${lines}" "${MAX_LINES}"
    violations=$((violations + 1))
  fi
done < <(measure)

if (( violations > 0 )); then
  cat >&2 <<EOF

File-size rule violated (${violations} file(s)).

dev-rules.md sets a ~${MAX_LINES}-line guideline, one responsibility per file.
Either split the file by responsibility, or — if the growth is genuinely
justified — refresh the baseline deliberately:

    scripts/check-file-size.sh --update

Refreshing the baseline is a conscious decision to take on debt, so it should be
visible in the diff and explained in the commit message.
EOF
  exit 1
fi

baselined=$(awk '$1 !~ /^#/ && NF == 2' "${baseline_file}" | wc -l | tr -d ' ')
echo "File-size check passed. ${baselined} file(s) baselined above ${MAX_LINES} lines; none grew, none added."
exit 0

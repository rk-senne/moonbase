#!/usr/bin/env bash
#
# Fail if any Go file is not gofmt-formatted.
#
# Formatting drift is the cheapest class of inconsistency to eliminate and the
# easiest to let rot, so it is checked mechanically rather than left to review.
#
# EXCEPTIONS
#
# The files listed in EXCEPTIONS below were already unformatted at HEAD *and*
# carry uncommitted work in progress. Formatting them would have meant either
# committing someone else's unreviewed changes or gambling with a `git stash pop`
# conflict against their work, so they are excepted instead.
#
# This list is a ratchet and should only ever shrink. When the work in progress on
# one of these files lands, run `gofmt -w` on it and delete its line here — the
# check prints a reminder when an excepted file is already clean, so a stale
# exception cannot hide.
#
# Usage:
#   scripts/check-gofmt.sh          # check (used by CI)
#   scripts/check-gofmt.sh --fix    # format everything, exceptions included
#
# Exit status: 0 when clean, 1 when an unexcepted file is unformatted.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

EXCEPTIONS="
internal/config/config.go
internal/pipeline/pipeline_new_test.go
internal/tui/app.go
internal/tui/coverage4_test.go
"

is_excepted() {
  local candidate="$1" e
  for e in ${EXCEPTIONS}; do
    [[ "${candidate}" == "${e}" ]] && return 0
  done
  return 1
}

# gofmt -l over tracked Go files. research/ holds extracted book text, not code.
unformatted() {
  gofmt -l . 2>/dev/null | grep -v '^research/' || true
}

if [[ "${1:-}" == "--fix" ]]; then
  files=$(unformatted)
  if [[ -z "${files}" ]]; then
    echo "Already formatted."
    exit 0
  fi
  echo "${files}" | while IFS= read -r f; do
    [[ -n "${f}" ]] || continue
    gofmt -w "${f}"
    echo "  formatted ${f}"
  done
  exit 0
fi

violations=0
dirty=$(unformatted)

if [[ -n "${dirty}" ]]; then
  while IFS= read -r f; do
    [[ -n "${f}" ]] || continue
    if is_excepted "${f}"; then
      continue
    fi
    printf '  UNFORMATTED  %s\n' "${f}"
    violations=$((violations + 1))
  done <<< "${dirty}"
fi

# Remind about exceptions that are no longer needed, so the list shrinks.
for e in ${EXCEPTIONS}; do
  [[ -f "${e}" ]] || continue
  if ! grep -qxF "${e}" <<< "${dirty}"; then
    printf '  NOTE: %s is now formatted — remove it from EXCEPTIONS in %s\n' \
      "${e}" "$(basename "${BASH_SOURCE[0]}")"
  fi
done

if (( violations > 0 )); then
  cat >&2 <<EOF

gofmt check failed (${violations} file(s)).

Fix with:
    scripts/check-gofmt.sh --fix
EOF
  exit 1
fi

excepted_count=$(printf '%s' "${EXCEPTIONS}" | grep -c . || true)
echo "gofmt check passed (${excepted_count} file(s) excepted pending in-progress work)."
exit 0

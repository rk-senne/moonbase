#!/usr/bin/env bash
#
# Fail if any Go file is not gofmt-formatted.
#
# Formatting drift is the cheapest class of inconsistency to eliminate and the
# easiest to let rot, so it is checked mechanically rather than left to review.
#
# EXCEPTIONS
#
# Empty, and it should stay that way. It previously held four files that were
# unformatted at HEAD while carrying uncommitted work; those landed formatted, so
# the list shrank to zero as intended.
#
# If an entry ever has to be added, treat it as temporary: the check prints a
# reminder once an excepted file is clean, so a stale exception cannot hide.
#
# Usage:
#   scripts/check-gofmt.sh          # check (used by CI)
#   scripts/check-gofmt.sh --fix    # format everything
#
# Exit status: 0 when clean, 1 when an unexcepted file is unformatted.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

EXCEPTIONS=""

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
echo "gofmt check passed."
exit 0

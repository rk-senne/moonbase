#!/usr/bin/env bash
#
# Extract plain text from the research PDFs for knowledge-base indexing.
#
# Why this exists: the knowledge indexer fails on some PDFs in research/.
# Two distinct failure modes were observed:
#
#   1. A malformed font resource ("Syntax Error: font resource is not a
#      dictionary") breaks PDF text extraction silently — the context
#      registers but contains zero searchable chunks.
#   2. Some PDFs register with Items: 1 yet every search against the returned
#      context ID fails with "Context not found", regardless of whether they
#      were indexed concurrently or strictly one at a time.
#
# Indexing the extracted .txt instead is reliable for all of them.
#
# Output is gitignored: the PDFs are the source of truth, this is derived.
#
# Usage:
#   scripts/extract-research-text.sh
#
# Requires: pdftotext (poppler).  macOS: brew install poppler
#           Debian/Ubuntu: apt install poppler-utils

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
research_dir="${repo_root}/research"
out_dir="${research_dir}/extracted-text"

if ! command -v pdftotext >/dev/null 2>&1; then
  echo "error: pdftotext not found. Install poppler (brew install poppler)." >&2
  exit 1
fi

if [[ ! -d "${research_dir}" ]]; then
  echo "error: ${research_dir} does not exist." >&2
  exit 1
fi

mkdir -p "${out_dir}"

# Books that need the text path. Keys are a distinctive filename prefix so the
# long z-library suffixes don't have to be repeated here.
titles=(
  "A Philosophy of Software Design|A Philosophy of Software Design (Ousterhout) - extracted text"
  "Designing Data-Intensive Applications|Designing Data-Intensive Applications (Kleppmann) - extracted text"
  "Monolith to Microservices|Monolith to Microservices (Sam Newman) - extracted text"
  "The Pragmatic Programmer|The Pragmatic Programmer (Thomas, Hunt) - extracted text"
)

extracted=0
for entry in "${titles[@]}"; do
  prefix="${entry%%|*}"
  dest="${entry##*|}"

  # Resolve the single PDF matching this prefix.
  shopt -s nullglob
  matches=("${research_dir}/${prefix}"*.pdf)
  shopt -u nullglob

  if (( ${#matches[@]} == 0 )); then
    echo "skip: no PDF matching '${prefix}*' in research/" >&2
    continue
  fi
  if (( ${#matches[@]} > 1 )); then
    echo "skip: '${prefix}*' matched ${#matches[@]} files, expected 1" >&2
    continue
  fi

  # pdftotext warns on malformed fonts but still extracts usable text.
  pdftotext "${matches[0]}" "${out_dir}/${dest}.txt" 2>/dev/null || true

  words=$(wc -w < "${out_dir}/${dest}.txt" | tr -d ' ')
  if [[ "${words}" -lt 1000 ]]; then
    echo "warn: ${dest} extracted only ${words} words — check the PDF" >&2
  else
    printf '%8s words  %s\n' "${words}" "${dest}"
    extracted=$((extracted + 1))
  fi
done

echo
echo "Extracted ${extracted} file(s) to research/extracted-text/"
echo "Index each with the knowledge tool, one at a time, and verify each is"
echo "searchable before relying on it."

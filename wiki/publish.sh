#!/usr/bin/env bash
# Publish the staged wiki/ pages to the moonbase GitHub Wiki.
# Prerequisite (one-time): create the first wiki page via the GitHub web UI
#   https://github.com/rk-senne/moonbase/wiki  → "Create the first page" → save.
# Then run this from the repo root: ./wiki/publish.sh
set -euo pipefail

REPO_SSH="git@github.com:rk-senne/moonbase.wiki.git"
SRC="$(cd "$(dirname "$0")" && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if ! git clone "$REPO_SSH" "$TMP" 2>/dev/null; then
  echo "❌ Could not clone $REPO_SSH"
  echo "   The wiki has no pages yet. Create the first page via the web UI, then re-run:"
  echo "   https://github.com/rk-senne/moonbase/wiki"
  exit 1
fi

# Copy every page except this script and the staging README.
for f in "$SRC"/*.md; do
  base="$(basename "$f")"
  [ "$base" = "README.md" ] && continue
  cp "$f" "$TMP/$base"
done

cd "$TMP"
if git diff --quiet && git diff --cached --quiet; then
  echo "✅ Wiki already up to date."
  exit 0
fi
git add -A
git commit -q -m "sync wiki from repo (wiki/)"
git push -q
echo "✅ Wiki published: https://github.com/rk-senne/moonbase/wiki"

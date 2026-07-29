# Changelog Discipline

## Location

`CHANGELOG.md` at repository root. Always updated as part of the change, not after.

## Format

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- feat(pipeline): retry logic for transient backend failures

### Fixed
- fix(tui): spinner not stopping after pipeline abort

### Changed
- refactor(agents): simplify frontmatter parsing to single pass
```

## Rules

- **Newest first** — latest release at the top.
- **Grouped by version and date** — use semver for releases.
- **One bullet per change** — prefix with conventional commit type and scope.
- **Record WHY, not just WHAT** — "retry logic for transient backend failures" beats "added retry".
- **Reference PR/branch** when known — helps trace decisions.
- **Unreleased section** — accumulate changes under `## [Unreleased]` until release.
- **No secrets** — never reference API keys, tokens, or internal URLs.
- **User-visible changes only** — skip internal refactors unless they change behavior.

## When to Update

| Change Type | Update Changelog? |
|---|---|
| New CLI command or flag | Yes |
| Bug fix | Yes |
| Performance improvement | Yes (if user-noticeable) |
| New agent or agent field | Yes |
| Internal refactor (no behavior change) | No |
| Test-only changes | No |
| Dependency update | Only if it fixes a bug or adds capability |

## Version Bumps

- **MAJOR** (X.0.0): breaking changes to CLI interface, agent format, or config
- **MINOR** (0.X.0): new features, new commands, new agents
- **PATCH** (0.0.X): bug fixes, performance improvements, doc fixes

## Releasing

1. Move `[Unreleased]` entries to new version heading with today's date.
2. Add comparison link at bottom: `[X.Y.Z]: https://github.com/.../compare/vPREV...vX.Y.Z`
3. Tag: `git tag vX.Y.Z && git push --tags`
4. goreleaser handles the rest.

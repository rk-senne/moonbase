# Wiki content (staged)

These Markdown files are the source for the moonbase **GitHub Wiki**, kept in-repo so
they're versioned, reviewable, and diffable alongside the code.

## Why staged here?

GitHub does **not** create a repository's `<repo>.wiki.git` until the **first page is
created through the web UI** — and there is no REST API to create wiki pages. So the wiki
can't be initialized purely from a script/CI. These files are ready to publish the moment
the wiki exists.

## Publish (one-time bootstrap, then repeatable)

1. **Bootstrap once (web UI):** open `https://github.com/rk-senne/moonbase/wiki`, click
   *Create the first page*, and save anything (it will be overwritten). This creates the
   `.wiki.git` repo.
2. **Publish these pages:**

   ```bash
   ./wiki/publish.sh          # from the repo root
   ```

   or manually:

   ```bash
   git clone git@github.com:rk-senne/moonbase.wiki.git /tmp/mb-wiki
   cp wiki/*.md /tmp/mb-wiki/
   cd /tmp/mb-wiki && git add -A && git commit -m "sync wiki from repo" && git push
   ```

## Pages

| File | Page |
|------|------|
| `Home.md` | Landing page |
| `Installation.md` | Install paths |
| `CLI-Reference.md` | Commands + flags |
| `The-Pipeline.md` | Phases, risk gate, adaptive depth, fan-out |
| `Agents.md` | Roster + agent file format |
| `Skills-and-Prompts.md` | Progressive skill loading |
| `Flywheel-and-Observability.md` | Learning insights + token/cost |
| `Configuration.md` | `config.yaml` reference |
| `Architecture.md` | Packages, stack, design |
| `Kiro-Native-Interop.md` | MCP + `moonbase compile` |
| `Contributing.md` | Dev workflow + quality gates |
| `_Sidebar.md` | Wiki navigation |

GitHub wiki link syntax is used for cross-page links (`[[Title|Page-Name]]`).

# Skills & Prompts

`moonbase init` scaffolds two knowledge stores in `.kiro/`:

- **`.kiro/skills/`** — domain knowledge operatives reference progressively
- **`.kiro/prompts/`** — reusable named workflows

## Progressive skill loading

Skills are Markdown files with YAML frontmatter. Moonbase loads only the **metadata** (name + description, ~100 tokens each) at startup, and injects the **full body on demand** — applying Anthropic's "metadata upfront, content on demand" guidance to save context-window tokens.

```markdown
---
name: docker-build
description: Docker multi-stage build patterns, layer caching, and CI integration. Use when working with Dockerfiles or container builds.
---

# Docker Build Patterns
## Multi-Stage Builds
...
```

**How it works:**

1. At discovery, the `SkillRegistry` reads only the first ~1 KB of each skill file to extract frontmatter — it does **not** read full bodies.
2. The composed prompt includes a lightweight catalog:

   ```
   --- AVAILABLE SKILLS ---
   Request any skill with @skill(name) to load its full content.
   | Skill | Description |
   | docker-build | Docker multi-stage build patterns… |
   --- END AVAILABLE SKILLS ---
   ```
3. When a task (or an operative) references `@skill(docker-build)`, the full body is loaded, frontmatter-stripped, truncated to the size cap, cached, and injected as a `--- SKILL: docker-build ---` block.

For ~10 skills of ~3 KB each, this cuts the skills section of the prompt from ~10K tokens to ~1K.

**Directory styles** (both supported):

```
.kiro/skills/
├── docker-build.md          ← flat file
├── git-workflow/
│   └── SKILL.md             ← directory-style (Kiro-native compatible)
└── legacy-no-frontmatter.md ← fallback: loaded eagerly, name from filename
```

Skills without frontmatter still work (loaded eagerly for backward compatibility). Emit Kiro-native `skill://` resource directories with `EmitKiroSkillResources` (used by the compile path).

## Prompts

Store reusable prompts as named snippets:

```bash
echo "Review this for OWASP Top 10 issues" | moonbase snippet save security-review
moonbase snippet list
```

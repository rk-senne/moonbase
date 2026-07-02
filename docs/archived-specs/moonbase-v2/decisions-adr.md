# Architecture Decisions: Moonbase

## ADR-001: Pure Markdown Agents with YAML Frontmatter

### Status
Accepted (2026-07-02)

### Context
Agents were originally defined as a triple: `config.json` (metadata) + `Profile.md` (backstory) + `{Name}.md` (operational prompt). This required file path resolution, had duplicated content (vault/ mirrored everything), and wasn't portable — you couldn't drop a single file into a project and have it work.

### Decision
Each agent is a single `.md` file. YAML frontmatter holds machine-readable metadata (tools, permissions, routing, hooks). The markdown body IS the system prompt. One file = one complete agent.

### Consequences
- Agents are portable — copy one file to any `.kiro/agents/` and it works
- No `file://` reference resolution needed
- Go code must parse YAML frontmatter from markdown (new parser needed)
- Old `.json` configs are deprecated and removed
- Doctrine content is embedded in each agent (no external references)

### Alternatives Considered
- **Keep JSON + separate .md:** Rejected — requires assembly, not portable
- **Pure YAML files:** Rejected — system prompts are long prose, YAML is terrible for that
- **TOML frontmatter:** Rejected — YAML is the established standard (Hugo, Jekyll, Obsidian, Kiro)

### Reversibility
HIGH — could revert to JSON by writing a converter script. Agent content wouldn't change.

---

## ADR-002: Kiro CLI as Primary Backend

### Status
Accepted (2026-07-02)

### Context
Moonbase originally planned support for 6 backends (kiro-cli, codex, openai, anthropic, ollama, clipboard). In practice, kiro-cli is the most capable because it handles tool execution, file access, and multi-turn conversation natively.

### Decision
Kiro CLI is the primary backend. The interface allows other backends, but pipeline execution targets kiro-cli specifically. Other backends get the composed prompt but may lack tool execution capability.

### Consequences
- Pipeline can rely on kiro-cli features (sub-agents, tool calls, file access)
- Users without kiro-cli get clipboard fallback (paste prompt into their tool)
- Backend interface stays generic for future expansion
- Pipeline quality depends on kiro-cli being installed

### Alternatives Considered
- **Backend-agnostic pipeline:** Rejected — lowest common denominator limits pipeline to prompt-in/text-out
- **API-first (OpenAI/Anthropic direct):** Rejected — requires API keys, rate limits, no tool execution
- **All backends equal:** Rejected — different capabilities make equal treatment impractical

### Reversibility
MEDIUM — backend interface is generic, adding first-class support for others is additive.

---

## ADR-003: Project Context Injection via Discovery

### Status
Accepted (2026-07-02)

### Context
Agents need to understand the project they're working in — its stack, conventions, existing specs, and rules. Without context, they produce generic work that doesn't match the project.

### Decision
A discovery phase runs before agent deployment. It looks for `.kiro/specs/`, `.kiro/steering/`, build configs, and README. Discovered context is prepended to the agent's prompt, giving it project awareness without the user repeating information.

### Consequences
- Agents automatically adapt to project conventions
- Steering rules are respected without manual loading
- Prompt grows with context — may hit token limits for large projects
- `inclusion: manual` in steering frontmatter allows opting out

### Alternatives Considered
- **User manually provides context:** Rejected — defeats the purpose, tedious
- **Knowledge base indexing:** Rejected — adds complexity, requires indexing step
- **Agent reads files during execution:** This still happens, but the preamble gives orientation

### Reversibility
HIGH — discovery is additive. Removing it just means agents get no preamble (they still work, just less context-aware).

---

## ADR-004: Embedded Operating Protocol (not referenced)

### Status
Accepted (2026-07-02)

### Context
Earlier design had agents reference external doctrine files (`file://../../doctrine/AgentOperatingProtocol.md`). This created a dependency chain — if a doctrine file moved, all agents broke.

### Decision
The Operating Protocol is embedded at the end of every agent's markdown body. Each agent is fully self-contained.

### Consequences
- Agents work without any external file resolution
- Updating the protocol requires updating all 14 files (or a script)
- Slight content duplication across agents (~50 lines each)
- Doctrine files in `doctrine/` remain as reference docs for humans, not loaded by agents

### Alternatives Considered
- **External references:** Rejected — breaks portability, requires resolution logic
- **Template inclusion at build time:** Rejected — adds build step, not pure markdown anymore
- **Single protocol file appended at runtime:** Rejected — adds runtime complexity, not self-contained

### Reversibility
HIGH — could add a build step that composes agents from parts if duplication becomes painful.

---

## ADR-005: Human Interaction Protocol (4-Level Uncertainty)

### Status
Accepted (2026-07-02)

### Context
AI agents either guess (and get things wrong) or ask too many questions (and slow everything down). Neither extreme is useful. The project needed a clear decision framework for when agents should ask vs proceed.

### Decision
4-level uncertainty model: CERTAIN (proceed), LIKELY (proceed + label assumption), UNCERTAIN (ask with structured question format), UNKNOWN (stop and ask). Embedded in every agent's Operating Protocol.

### Consequences
- Agents have a clear decision framework for ambiguity
- Users get focused questions with options and defaults (not vague "what should I do?")
- Non-blocking questions let work continue with defaults
- The protocol is advice, not enforcement — AI models follow it as strongly as any other prompt instruction

### Alternatives Considered
- **Always ask:** Rejected — too slow, annoying, blocks progress
- **Never ask:** Rejected — silent failures, wrong work, wasted effort
- **Binary ask/don't-ask:** Rejected — too coarse, misses the "proceed with labelled assumption" middle ground

### Reversibility
HIGH — it's prompt content, can be adjusted per-agent or removed entirely.

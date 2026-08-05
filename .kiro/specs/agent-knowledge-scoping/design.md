# Design: Agent Knowledge & Capability Scoping

## Architecture Decision (ADR-summary)

**Decision:** Introduce a **four-layer knowledge model** and make both **skills**
and **MCP servers** *per-operative scoped*, reusing the `progressive-skills`
loading engine. Knowledge that is *principle* stays in steering (L0); *procedure*
becomes on-demand skill cards (L1/L2); *source text* is reached by retrieval (L3).
Live external capability is granted via **least-privilege, per-agent MCP**.

**Why:** Global skill catalogues are noise; raw-PDF injection exhausts context;
ungoverned MCP is an unbounded attack surface. Scoping raises signal-to-noise,
cost, and safety without changing the loading engine. (Anthropic Agent Skills
five habits; MCP "trust boundary + least privilege"; doctrine "start simple.")

**Consequences:** two new optional frontmatter fields (`skills`,
`skills_strict`), one extension to `MCPServerConfig` (`read_only`,
`justification`), a composer filter, lint rules, five skill cards, one shared
retrieval MCP, and a governance matrix. All backward compatible.

**Reversibility:** every field is optional and additive; deleting them restores
today's behaviour. Blast radius = composer catalogue section + lint. High
reversibility.

---

## The Four-Layer Knowledge Model

```
L0  Steering / Doctrine     ── always in prompt ──  universal PRINCIPLES
        (doctrine/*.md, .kiro/steering/*.md)        (Clean Code law, Arch rules)
                │  injected by ComposePrompt for every agent
                ▼
L1  Skill Catalogue         ── always, but SCOPED per agent (this spec, §B) ──
        (SkillRegistry.List() filtered by agent.Skills)   ~100 tokens/skill
                │  agent references @skill(name)
                ▼
L2  Skill Body              ── on demand (progressive-skills engine) ──
        (SkillRegistry.LoadContent) distilled, book-grounded, 1–10K tokens
                │  card insufficient → needs source text
                ▼
L3  Retrieval (RAG)         ── on demand, scoped to agents that need it (§D) ──
        knowledge tool / shared read-only MCP over `moonbase-research-books`
```

Placement rule (documented for authors): **principle → L0, procedure → L1/L2,
source passage → L3.** Never inline raw books (AC-A.3).

---

## Files Affected

| File | Change | Purpose |
|------|--------|---------|
| `internal/agents/agent.go` | modify | Add `Skills []string`, `SkillsStrict bool`; extend `MCPServerConfig` with `ReadOnly bool`, `Justification string` |
| `internal/discovery/compose.go` | modify | Filter the skill catalogue by `agent.Skills`; honour `skills_strict` at resolution |
| `internal/discovery/skills.go` | reuse | `SkillRegistry` from progressive-skills (List/Get/LoadContent) — no change |
| `internal/agents/validate.go` (or lint path) | modify | Validate `skills` resolvable, MCP well-formed, `read_only` default, no secret literals |
| `internal/compile/compile.go` | reuse/verify | Already emits `mcpServers`; confirm per-agent isolation; do not emit governance-only fields |
| `skills/` (or `.kiro/skills/`) | new | 5 book-distilled skill cards |
| `doctrine/` + `.kiro/steering/` | modify | Add "MCP least-privilege" section; record placement rule |
| `docs/agent-format.md` | modify | Document new fields |
| `.kiro/specs/agent-knowledge-scoping/decisions-adr.md` | new | ADR |
| `*_test.go` | new/modify | Composer scoping, parsing, strict/soft, MCP emission |

---

## Component Designs

### §B — Per-Agent Skill Scoping

**Frontmatter (additions to `Agent`):**
```go
Skills       []string `yaml:"skills,omitempty"`        // catalogue allowlist (skill names)
SkillsStrict bool     `yaml:"skills_strict,omitempty"` // true = out-of-scope @skill refused
```

Agent `.md` example (Numbuh 0, Architect):
```yaml
skills:
  - deep-modules
  - diagram-notation
  - clean-architecture
skills_strict: false   # soft scope (default): @skill can still reach others
```

**Composer filter (`compose.go`):**
```
catalogue = registry.List()
if len(agent.Skills) > 0:
    allow = set(expandGroups(agent.Skills))          // AC-B.4 group expansion (optional)
    catalogue = [s for s in catalogue if s.Name in allow]
    for name in agent.Skills where registry.Get(name)==nil:
        warn("agent %s scopes unknown skill %q", agent.Name, name)   // AC-B.1
emit "--- AVAILABLE SKILLS ---" from catalogue         // scoped view
```
- Absent `skills:` → `catalogue` unfiltered = today's behaviour (AC-B.2).
- `@skill(name)` resolution (in the progressive-skills hook): if `skills_strict`
  and `name ∉ agent.Skills` → refuse with `"skill %q is out of scope for %s"`
  (AC-B.3); else resolve normally (soft).

**Groups (optional, AC-B.4):** a `skills-groups.yaml` maps
`clean-code → [error-handling, refactoring-safely, code-review]`. `@group(x)` in
`skills:` expands at load. Schema tolerates it now; implementation may defer.

### §A — Research → Skill Cards

**Card format** (progressive-skills compatible + `source`):
```markdown
---
name: deep-modules
description: >
  Ousterhout's deep-vs-shallow module test and complexity signals. Use when
  designing a module boundary, reviewing an interface, or judging if an
  abstraction earns its keep.
source: "A Philosophy of Software Design (John Ousterhout)"
---

# Deep Modules

A module is *deep* when a simple interface hides a large implementation…
## The test
## Red flags (shallow modules, information leakage, temporal decomposition)
## Applying it in review
```
- **Scope tightly** (one theme/book, not the whole book), **1–10K tokens**
  (on-demand tier), **explicit "use when"** in `description` (AC-A.1).
- **`source` is mandatory** — the card is a distillation, and L3 retrieval can
  fetch the exact passage from that book if needed.

**Initial card set (AC-A.2) — language-agnostic craft, and agent mapping:**

| Skill card | Source | Primary agents |
|-----------|--------|----------------|
| `deep-modules` | Ousterhout, *A Philosophy of Software Design* | N0, N2, N3 |
| `diagram-notation` | Brown, *The C4 Model* | N0, N2, N999 |
| `reviewing-changes` | Martin, *The Clean Coder* | N4, N5 |
| `refactoring-safely` | Fowler / Martin | N3, N86 |
| `testing-discipline` | Martin, *Clean Code* | N4, N13 |
| `root-cause-analysis` | Hunt/Thomas, *The Pragmatic Programmer* | N4, N13, Sector-Z |
| `reading-unfamiliar-code` | code archaeology | N3, N86, Sector-Z |
| `decision-records` | Architecture Doctrine (ADR) | N0, N1, N2, N5 |
| `writing-clear-docs` | docs-grounded-in-code | N999 |
| `threat-modeling` | attacker mindset (STRIDE-lite) | N274 |
| `devops-flow` | Kim et al., *The Phoenix Project* | N362, N0 |
| `estimation-and-scope` | Martin, *The Clean Coder* | N1 |

These are **language/stack-agnostic** (AC-A.4): craft and judgment that transfer
across any repo. Existing stack-flavoured cards (`concurrency-patterns`,
`docker-build`, `api-design`, `git-workflow`, `observability`, `error-handling`,
`security-review`, `code-review`) remain available in the registry but are
**opt-in per project**, not part of the default per-agent scopes.

### §C — Per-Agent MCP Governance

**Frontmatter (`MCPServerConfig` already exists; add governance fields):**
```go
type MCPServerConfig struct {
    Name         string            `yaml:"name"`
    Command      string            `yaml:"command"`
    Args         []string          `yaml:"args,omitempty"`
    Env          map[string]string `yaml:"env,omitempty"`          // VALUES must be env-var refs, not literals
    AllowedTools []string          `yaml:"allowed_tools,omitempty"` // per-server tool scope (already present)
    ReadOnly     bool              `yaml:"read_only,omitempty"`     // NEW — governance default true (AC-C.2)
    Justification string           `yaml:"justification,omitempty"` // NEW — why this agent needs it (AC-C.4)
}
```
- `ReadOnly`, `Justification` are **moonbase governance metadata** (used by lint +
  the matrix + review). Kiro's `KiroMCPServer` has only Command/Args/Env, so these
  are **not emitted** downstream — they govern *assignment*, not runtime. Note in
  compile: strip governance-only fields when producing `KiroMCPServer`.
- **Separation of duties:** to grant both read and write, declare **two** server
  entries (`foo-ro` read_only, `foo-rw`) rather than one dual-capability server
  (Google MCP guidance). The schema already permits multiple entries.
- **Least privilege default:** validation treats missing `read_only` as `true`
  and requires an explicit `read_only: false` + `justification` to grant writes
  (fail safe, AC-C.2/C.5).

**Default MCP palette — maintained, language-agnostic reference servers (AC-C.6):**

| Server | Capability | Default scope |
|--------|-----------|---------------|
| `filesystem` | scoped file read/write | **workspace dir only** |
| `git` | history, blame, diff, commit search | **read-only** |
| `fetch` | pull + convert web/doc content | read-only (egress) |
| `memory` | knowledge-graph persistence across a mission | local |
| `sequential-thinking` | structured multi-step reasoning | local, no I/O |
| `time` | timezone / date math | local |
| `research-kb` *(shared)* | L3 retrieval over `moonbase-research-books` | **read-only, single shared server** (AC-C.3) |

Platform/product servers (**GitHub/GitLab**, **web-search** (Brave/Exa),
**Playwright**, **Sentry**, databases) are **opt-in per project/agent with
justification** (AC-C.6) — never a default. `everything` is dev/test only.

**Per-operative governance matrix (default scopes; all least-privilege, RO unless noted):**

| Operative | Skills (scoped) | MCP servers |
|-----------|-----------------|-------------|
| N0 Architect | deep-modules, diagram-notation, decision-records | git, memory, sequential-thinking |
| N1 Analyst | estimation-and-scope, decision-records | fetch, memory |
| N2 Design | deep-modules, decision-records, diagram-notation | git, sequential-thinking |
| N3 Implementer | refactoring-safely, testing-discipline, reading-unfamiliar-code | filesystem (workspace), git |
| N4 QA | testing-discipline, reviewing-changes, root-cause-analysis | filesystem (RO), sequential-thinking |
| N5 Reviewer | reviewing-changes, decision-records | git |
| N13 Chaos | root-cause-analysis, testing-discipline | sequential-thinking, time |
| N86 Tech-Debt | refactoring-safely, reading-unfamiliar-code | git, filesystem (RO) |
| N274 Security | threat-modeling | git, fetch *(+ web-search opt-in)* |
| N362 DevOps | devops-flow | time, git *(+ GitHub opt-in)* |
| N999 Docs | writing-clear-docs, diagram-notation | fetch, filesystem |
| Sector-Z | reading-unfamiliar-code, root-cause-analysis | **git** (core) |
| *all (opt-in)* | — | `research-kb` (shared, RO) |

`research-kb` is **one shared read-only** server (AC-C.3) — not duplicated per
agent. Static book knowledge lives in skills (L1/L2) + this single retrieval
path; no bespoke per-agent knowledge servers. Separation of duties: to grant
writes, declare a **second** server entry (`*-rw`, read_only:false, justified)
rather than widening a read-only one.

### §C.1 — Kiro-native compatibility (AC-C.7)

The compiled MCP config already matches Kiro's schema
(`KiroMCPServer{Command, Args, Env}` in `internal/compile/types.go`), which is
exactly Kiro's `mcp.json` shape. Consequences:
- An operative's `mcp_servers:` compiles to the same JSON Kiro loads from a
  workspace/user `mcp.json` (`command`/`args`/`env`), so servers behave
  identically whether launched by moonbase or a Kiro workspace.
- Kiro CLI v3 "inline agent" config (MCP servers embedded in the agent markdown)
  maps 1:1 to moonbase frontmatter — no divergence to maintain.
- Reference servers run via their standard launchers, e.g.
  `npx -y @modelcontextprotocol/server-filesystem <dir>` or
  `uvx mcp-server-git --repository <repo>`; `env` values are var-refs only
  (AC-C.5). Governance-only fields (`read_only`, `justification`) are moonbase
  metadata and are **stripped** before emitting `KiroMCPServer` (AC-C.1).

### §D — Retrieval tier wiring
- Agents that benefit from L3 (N0, N4, N274 at minimum) explicitly carry the
  retrieval capability (the `knowledge` tool today, or the shared `research-kb`
  MCP). Presence is intentional and listed, not assumed globally (AC-D.2).
- Retrieval returns *passages*, cited back to the `source` book — closing the
  loop with the L2 cards.

---

## Validation & Lint (AC-E.2, AC-C.5)

`moonbase lint` (per-agent) checks:
1. every `skills:` entry resolves in the registry (else warn, drop);
2. each `mcp_servers[]` has non-empty `name` + `command`; unknown keys rejected;
3. `read_only` defaults true; `read_only: false` **requires** `justification`;
4. `env` values match an env-var reference form (e.g. `${VAR}` or `VAR`), never a
   secret-looking literal (high-entropy / `sk-…` / `AKIA…`) → fail closed;
5. all 14 shipped agents pass unchanged (they declare neither field yet).

---

## Data Flow (composition, scoped)

```
ComposePrompt(agent, ctx, task)
  ├─ L0: steering + doctrine (unchanged)
  ├─ L1: registry.List()  ──filter by agent.Skills──▶ scoped "AVAILABLE SKILLS"
  ├─ (agent body may @skill(name))
  │      └─ progressive-skills hook → strict? in-scope? → LoadContent → L2 inject
  └─ compile: agent.MCPServers ──governance strip──▶ KiroAgent.mcpServers (per-agent)
```

---

## Testing (AC-E.3)

| Test | Scenario |
|------|----------|
| `TestCompose_SkillCatalogue_ScopedByAgent` | agent with `skills:[a,b]` sees only a,b |
| `TestCompose_SkillCatalogue_NoField_FullCatalogue` | absent field = all skills (back-compat) |
| `TestCompose_UnknownScopedSkill_WarnsAndDrops` | `skills:[ghost]` → warning, not in catalogue |
| `TestSkillResolve_Strict_RefusesOutOfScope` | `skills_strict:true` + `@skill(x∉scope)` → refusal |
| `TestSkillResolve_Soft_ResolvesOutOfScope` | default soft → resolves |
| `TestParseAgent_SkillsAndMCPFields` | frontmatter parses new fields |
| `TestValidate_MCP_ReadOnlyDefaultTrue` | missing read_only ⇒ true |
| `TestValidate_MCP_WriteRequiresJustification` | read_only:false w/o justification ⇒ error |
| `TestValidate_MCP_RejectsSecretLiteral` | inlined `sk-…`/`AKIA…` ⇒ error |
| `TestCompile_MCP_PerAgentIsolation_StripsGovernance` | only that agent gets its servers; read_only/justification not emitted |

All table-driven where 3+ cases; `go test -race ./...` green; `moonbase lint` green.

---

## Migration Strategy (phased, non-breaking)

1. **Phase 1 — schema + composer:** add fields (optional), composer scoping,
   lint. Zero agents scoped yet → no behaviour change. Ship + test.
2. **Phase 2 — cards:** author the 5 book skill cards (+ `source`). Available to
   all (still unscoped). Ship + test.
3. **Phase 3 — scope agents:** add `skills:` allowlists + `mcp_servers:` (RO,
   justified) to the 14 agents per the matrix. Record ADR + governance matrix.
4. **Phase 4 — retrieval:** define the shared read-only `research-kb` MCP;
   attach L3 to N0/N4/N274; document the placement rule.

Each phase independently deployable + reversible (doctrine: small phased change).

---

## Security Considerations
- MCP servers are trust boundaries: least privilege, separation of duties,
  env-var-only secrets, fail-closed validation (AC-C.2/C.5).
- Skill/agent content remains untrusted text in the prompt (unchanged model).
- Scoping reduces each operative's blast radius (fewer tools/knowledge than the
  union) — a security *improvement*, aligned with the guardrails doctrine.

# Requirements: Agent Knowledge & Capability Scoping

## Overview

Moonbase operatives share one flat, global pool of knowledge and capability:
every agent sees the same skill catalogue, and the research library
(`research/*.pdf`, indexed as the `moonbase-research-books` knowledge base) is
distilled only into always-on `doctrine/` + `.kiro/steering/` prose. Two gaps
follow:

1. **Knowledge is not packaged for retrieval.** The research books exist as
   (a) big PDFs behind a `knowledge` tool and (b) hand-distilled doctrine. There
   is no middle tier — *actionable, book-grounded skill cards* an operative can
   pull on demand for the task in front of it.
2. **Knowledge and capability are not scoped per operative.** Skills are a global
   catalogue (confirmed: no `skills:` field in agent frontmatter — every agent
   sees every skill). MCP servers are supported by the compiler
   (`internal/compile/types.go: MCPServers`) but there is no governance for
   *which* operative gets *which* server, nor least-privilege discipline.

This spec adds a **layered knowledge model** and **per-operative scoping** for
both skills and MCP servers, so each operative carries only the knowledge and
live capability its role needs — improving signal-to-noise, cost, and safety.

> **Relationship to `progressive-skills`:** That spec builds the *loading
> mechanism* (L1 metadata catalogue → L2 on-demand body → L3 resources, the
> `SkillRegistry`, and the `@skill(name)` protocol). This spec **consumes** that
> mechanism and adds three things it does not cover: (A) the *content* —
> research-distilled skill cards; (B) *per-agent skill scoping*; (C) *per-agent
> MCP governance*. `progressive-skills` is a hard dependency (see Dependencies).

> **AC-ID convention:** stable IDs `AC-{section}.{index}`, sections A–E. IDs are
> stable — do not renumber; append with the next index.

---

## Research Grounding

This design applies current, cited best practice:

- **Agent Skills / progressive disclosure** (Anthropic, released as an open
  standard Dec 2025; adopted by OpenAI, Google, GitHub, Cursor). Three-level
  architecture *metadata → core content → resources*; five authoring habits:
  **scope tightly, write explicit activation descriptions, use progressive
  disclosure, chain small skills, monitor in production**. Size tiers (LangChain,
  *Skills* tutorial): `<1K` tokens may inline, `1–10K` load on demand, `>10K`
  paginate/search. A skill is "a retrieval unit … not necessarily backed by
  embeddings."
- **Knowledge tiering** (RAG vs fine-tune vs prompt): the industry "ladder" —
  prompt/skills for procedural knowledge, **RAG for grounding in source text**,
  fine-tune only for durable behavioural change. Fine-tuning is **out of scope**
  (cost + no behavioural need). Our books map to *skills* (distilled) + *RAG*
  (raw retrieval), never weights.
- **MCP governance** (Google Spanner MCP guidance; OWASP/AgentsID 2026 surveys):
  MCP is "vulnerable by default" (auth delegated to transport, no per-tool authz,
  no standard audit format). Therefore: **treat every MCP server as a first-class
  trust boundary; enforce least privilege; separate duties** (distinct
  read-only vs write credentials); inventory + monitor permission inheritance.
- **MCP reference servers + Kiro compatibility** (verified against
  `modelcontextprotocol/servers` and `kiro.dev/docs`): the *maintained*,
  language-agnostic reference set is **filesystem, git, fetch, memory,
  sequential-thinking, time, everything** (GitHub/Slack/Postgres/etc. are now
  archived or standalone). Kiro loads MCP via `mcp.json`
  (`command`/`args`/`env`/`disabled`/`autoApprove`) at workspace or user level,
  and **Kiro CLI v3 embeds MCP servers + tool-category tags inline in the agent
  markdown** — identical to moonbase's `mcp_servers:` frontmatter, so no new
  plumbing is required.
- **Moonbase's own `AgentEngineeringDoctrine`**: three components (model, tools,
  instructions); *tools expand what an agent CAN do, instructions constrain what
  it SHOULD do*; **start simple, add complexity only when tool count exceeds
  clarity**; every dependency is a trust decision + attack surface.

---

## User Stories

- **US-1 (Research → Skills):** As a maintainer, I want the research books
  distilled into small, book-grounded skill cards so operatives get actionable
  expertise on demand without me pasting book text into prompts.
- **US-2 (Per-agent skills):** As an operative, I want to see only the skills
  relevant to my role so my catalogue is signal, not noise.
- **US-3 (Deep retrieval):** As an operative facing a question my skill card
  doesn't cover, I want to retrieve the exact passage from the research KB.
- **US-4 (Per-agent MCP):** As a maintainer, I want to grant an operative a live
  capability (e.g. dependency-CVE lookup for Security) without granting it to
  everyone, under least-privilege.
- **US-5 (Governance):** As a reviewer, I want an auditable matrix of which
  operative has which skills and MCP servers, and why.

---

## Acceptance Criteria

### A. Research → Skill Cards

#### AC-A.1 — Book-distilled skill card format
- **WHEN** a research book is distilled into a skill
- **THEN** it is authored as a `progressive-skills`-compatible markdown file with
  YAML frontmatter (`name`, `description`) plus a `source` field crediting the
  book (title + author)
- **SHALL** keep the body in the **1–10K token** on-demand tier (scope tightly;
  one book-theme per card, not the whole book)
- **SHALL** state, in the `description`, *when to use it* (explicit activation),
  per Anthropic authoring guidance.

#### AC-A.2 — Initial card set (language-agnostic craft)
- **WHEN** this spec is implemented
- **THEN** at minimum these **language-agnostic** cards exist under
  `.kiro/skills/` (or `skills/`), each grounded in a named research source:
  `deep-modules` (Ousterhout, *A Philosophy of Software Design*),
  `diagram-notation` (Brown, *The C4 Model*),
  `reviewing-changes` (Martin, *The Clean Coder* — four-lens review),
  `refactoring-safely` (Fowler/Martin — behaviour-preserving, small steps),
  `testing-discipline` (Martin, *Clean Code* — FIRST, naming, what-not-to-test),
  `root-cause-analysis` (Hunt/Thomas, *The Pragmatic Programmer* — reproduce →
  bisect → 5-whys),
  `reading-unfamiliar-code` (code archaeology — entry points, tracing, git as
  evidence),
  `decision-records` (Architecture Doctrine — ADR
  context/decision/consequences/reversibility),
  `writing-clear-docs` (docs grounded in code state),
  `threat-modeling` (attacker mindset, STRIDE-lite — no language),
  `devops-flow` (Kim et al., *The Phoenix Project* — flow, WIP limits, incident
  response),
  `estimation-and-scope` (Martin, *The Clean Coder* — reduce scope, never skip
  tests).
- **SHALL** each cite its source and be ≤ the on-demand size tier.
- **SHALL** favour portable engineering *craft/judgment* over stack mechanics
  (existing stack-flavoured cards such as `concurrency-patterns`, `docker-build`,
  `api-design` are retained but NOT part of the default per-agent scopes).

#### AC-A.4 — Language-agnostic by default
- **WHEN** a skill card or MCP server is added to the default per-agent scope
- **THEN** it SHALL be language/stack-agnostic — leverageable across any project
  regardless of programming language or framework
- **SHALL** treat stack-specific knowledge (a given language's concurrency model,
  a specific build tool, a specific database) as *opt-in per project*, never a
  default operative scope. Rationale: operatives run against many repos; their
  baseline knowledge/capability must transfer.

#### AC-A.3 — No raw-PDF injection
- **WHEN** knowledge is embedded into an agent prompt
- **THEN** it is a distilled card (L1/L2) or a retrieved passage (L3) — **never**
  a raw PDF dump
- **SHALL NOT** add the PDFs or their full extracted text to any prompt or
  agent file (context-exhaustion + noise; the doctrine truncates for this reason).

### B. Per-Agent Skill Scoping

#### AC-B.1 — `skills:` frontmatter allowlist
- **WHEN** an agent `.md` frontmatter declares a `skills:` list of skill names
- **THEN** only those skills appear in that agent's catalogue section of the
  composed prompt
- **SHALL** validate each name against the registry; unknown names are dropped
  with a structured warning (do not crash).

#### AC-B.2 — Absent field = backward compatible
- **WHEN** an agent has **no** `skills:` field
- **THEN** behaviour is unchanged from `progressive-skills` (the agent sees the
  full catalogue) — this field is **opt-in**, zero breaking change.

#### AC-B.3 — Scoping is catalogue-only, not a hard gate by default
- **WHEN** an agent is scoped to a subset
- **THEN** the catalogue it sees is filtered, but an explicit `@skill(name)` for
  an out-of-scope but existing skill still resolves (soft scope), **unless**
  `skills_strict: true` is set, in which case out-of-scope requests are refused
  with a clear message (hard scope).
- **SHALL** default to soft scope (`skills_strict` defaults false).

#### AC-B.4 — Wildcard / group support
- **WHEN** `skills:` contains a group token (e.g. `@group(clean-code)`)
- **THEN** it expands to the group's member skills
- **SHALL** support at least explicit names; groups are OPTIONAL (design may
  defer, but the schema must not preclude them).

### C. Per-Agent MCP Governance

#### AC-C.1 — Per-agent MCP declaration compiles
- **WHEN** an agent frontmatter declares `mcp_servers:` (name, command/url,
  scope, read_only)
- **THEN** `moonbase compile` emits them into that agent's Kiro `mcpServers`
  (already supported by `internal/compile`), and **only** that agent's compiled
  config contains them
- **SHALL** validate required fields and reject malformed entries with a clear
  error (fail closed).

#### AC-C.2 — Least privilege + separation of duties
- **WHEN** an MCP server is assigned to an operative
- **THEN** it is granted with the **minimum** scope needed for that role, and
  read vs write capabilities use **separate** server entries/credentials where
  the backend supports it (per Google MCP guidance)
- **SHALL** default `read_only: true` when unspecified (fail safe).

#### AC-C.3 — Shared retrieval MCP over the research KB
- **WHEN** an operative needs source-text retrieval (L3)
- **THEN** it uses a **single, shared, read-only** retrieval MCP (or the existing
  `knowledge` tool) pointed at `moonbase-research-books` — **not** a bespoke
  per-agent knowledge server
- **SHALL** document that static knowledge belongs in skills/RAG, not in N
  duplicated MCP servers.

#### AC-C.4 — Governance matrix + doctrine
- **WHEN** MCP servers are assigned
- **THEN** a governance matrix (operative → servers → scope → read/write →
  justification) is recorded in the design doc, and the security doctrine gains a
  short "MCP least-privilege" section
- **SHALL** treat every server as a trust boundary (Non-Negotiable: every
  dependency is an attack surface).

#### AC-C.5 — No secrets in agent files
- **WHEN** an MCP server needs credentials
- **THEN** they are referenced via env-var names, never inlined
- **SHALL** fail validation if a value looks like a secret literal (Non-Negotiable
  #1: secrets are sacred).

#### AC-C.6 — Default palette is the language-agnostic reference set
- **WHEN** an operative is granted MCP capability by default
- **THEN** it SHALL be drawn from the maintained, language-agnostic reference
  servers: **filesystem** (scoped to workspace), **git** (read-only), **fetch**,
  **memory**, **sequential-thinking**, **time** — plus the shared read-only
  retrieval server (AC-C.3)
- **SHALL** treat platform/product servers (GitHub/GitLab, web-search, Playwright,
  Sentry, databases) as *opt-in per project/agent with justification*, never a
  default scope. `everything` is dev/test only.

#### AC-C.7 — Kiro-native compatibility
- **WHEN** an agent with `mcp_servers:` is compiled
- **THEN** the emitted config SHALL match Kiro's `mcp.json` shape
  (`command`/`args`/`env`) so it loads identically whether run via moonbase or a
  Kiro workspace/user config
- **SHALL** confirm parity with Kiro CLI v3 inline-agent MCP embedding (the
  frontmatter → compiled JSON path already exists in `internal/compile`).

### D. Layered Knowledge Model (retrieval tier)

#### AC-D.1 — Four explicit layers
- **WHEN** knowledge is provided to an operative
- **THEN** it resolves through explicit layers:
  **L0 Steering/Doctrine** (always-on principles) →
  **L1 Skill catalogue** (metadata, scoped per B) →
  **L2 Skill body** (on-demand via `@skill`) →
  **L3 Retrieval** (RAG/knowledge tool over the research KB)
- **SHALL** document which layer a given kind of knowledge belongs in
  (principles→L0, procedures→L1/L2, source passages→L3).

#### AC-D.2 — Retrieval tool scoped to agents that need it
- **WHEN** an agent's role benefits from source retrieval (e.g. N0, N4, N274)
- **THEN** the `knowledge`/retrieval capability is explicitly present for those
  agents and intentional (not incidental)
- **SHALL NOT** be assumed globally just because the tool is listed.

### E. Cross-Cutting (evidence, safety, tests)

#### AC-E.1 — Backward compatibility
- All new frontmatter fields (`skills`, `skills_strict`, `mcp_servers`) are
  OPTIONAL; agents without them behave exactly as today.

#### AC-E.2 — Validation & lint
- `moonbase lint` validates the new fields (skill names resolvable, MCP entries
  well-formed, no secret literals) and passes for all 14 shipped agents.

#### AC-E.3 — Tests
- Composer scoping (filtered catalogue), frontmatter parsing (skills/mcp),
  strict vs soft scope, and MCP compile emission each have unit tests
  (`TestFn_Scenario` naming; table-driven where 3+ cases). `go test -race ./...`
  green.

#### AC-E.4 — Docs & ADR
- An ADR records the layered-knowledge decision and the "skills for knowledge,
  MCP for live capability" boundary. The agent-format doc documents the new
  fields.

---

## Scope

**In scope:** `skills:`/`skills_strict:` frontmatter + composer catalogue
scoping; per-agent `mcp_servers:` governance + validation + least-privilege
defaults; the initial 5 book-distilled skill cards; the four-layer knowledge
model documentation; a shared read-only retrieval MCP definition; the
operative→skill/MCP mapping matrix; ADR + agent-format doc updates; tests.

**Out of scope:** the skill *loading* engine (owned by `progressive-skills`);
fine-tuning; a skill marketplace/remote fetch; building the MCP servers'
backends themselves (we define assignment + governance, not the CVE service);
changing the pipeline/risk-gate.

---

## Dependencies

| Dependency | Status | Impact |
|-----------|--------|--------|
| `progressive-skills` spec | exists (registry implemented in `skills.go`) | Hard dep — provides L1/L2 loading + `@skill` protocol this spec scopes |
| `internal/compile` MCPServers | exists (`types.go`, `compile.go`) | Per-agent MCP already compiles; this adds governance + validation |
| `moonbase-research-books` KB | indexed | L3 retrieval source for cards + shared MCP |
| `gopkg.in/yaml.v3` | in go.mod | frontmatter parsing for new fields |
| doctrine/steering | exists | L0 layer; gains an MCP least-privilege section |

---

## Risks

| Risk | Mitigation |
|------|-----------|
| Over-scoping starves an agent of a needed skill | Soft scope by default (AC-B.3); `@skill` still resolves |
| MCP sprawl (N servers, N attack surfaces) | AC-C.3 shared retrieval MCP; least-privilege defaults; governance matrix |
| Distilled cards drift from the source books | AC-A.1 `source` citation; cards reviewed against the KB; keep small |
| Secret leakage via MCP config | AC-C.5 env-var-only + lint refusal of literals |
| Breaking existing agents | AC-E.1 all fields optional; AC-B.2 absent = full catalogue |
| Raw-PDF temptation | AC-A.3 explicit prohibition; L3 retrieval is the sanctioned path |

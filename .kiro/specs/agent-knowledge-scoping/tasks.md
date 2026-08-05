# Tasks: Agent Knowledge & Capability Scoping

Phased, non-breaking delivery. Each task is a single verifiable increment:
implement → `go build ./... && go vet ./... && go test -race ./...` green →
`moonbase lint` green → one conventional commit. Cross-references
`requirements.md` (AC-x.y) + `design.md` (§/ADR). Effort: *S*<½d, *M*~1d, *L*>1d.

> **Hard dependency:** `progressive-skills` provides the `SkillRegistry` +
> `@skill(name)` loading engine. If a needed hook is not yet present, land the
> minimal part of it first (note it in the task) — do not reimplement loading.

---

## Phase P1 — Schema + Composer Scoping (§B)

- [ ] **T1.1** Add `Skills []string` + `SkillsStrict bool` to `Agent`
  (`internal/agents/agent.go`); parse + round-trip in the loader. — AC-B.1/B.2 — *S*
- [ ] **T1.2** Composer: filter the AVAILABLE SKILLS catalogue by `agent.Skills`
  (absent ⇒ full catalogue); warn+drop unknown names. — AC-B.1/B.2 — *M*
- [ ] **T1.3** `@skill(name)` resolution honours `skills_strict` (strict ⇒ refuse
  out-of-scope with clear message; soft ⇒ resolve). — AC-B.3 — *S*
- [ ] **T1.4** Tests: `TestCompose_SkillCatalogue_ScopedByAgent`,
  `_NoField_FullCatalogue`, `_UnknownScopedSkill_WarnsAndDrops`,
  `TestSkillResolve_Strict_RefusesOutOfScope`, `_Soft_ResolvesOutOfScope`,
  `TestParseAgent_SkillsAndMCPFields`. — AC-E.3
- [ ] **P1 gate:** build/vet/test-race green; `moonbase lint` green; all 14
  shipped agents unchanged (no `skills:` yet).

## Phase P2 — MCP Governance + Lint (§C)

- [ ] **T2.1** Extend `MCPServerConfig` with `ReadOnly bool` +
  `Justification string`; parse. — AC-C.1 — *S*
- [ ] **T2.2** Compile: confirm per-agent `mcpServers` isolation; **strip**
  governance-only fields (`ReadOnly`/`Justification`) from `KiroMCPServer`. — AC-C.1 — *S*
- [ ] **T2.3** Lint rules: MCP `name`+`command` required; `read_only` default
  true; `read_only:false` requires `justification`; `env` values must be
  var-refs (reject secret literals, fail closed). — AC-C.2/C.5/E.2 — *M*
- [ ] **T2.4** Tests: `TestValidate_MCP_ReadOnlyDefaultTrue`,
  `_WriteRequiresJustification`, `_RejectsSecretLiteral`,
  `TestCompile_MCP_PerAgentIsolation_StripsGovernance`. — AC-E.3
- [ ] **P2 gate:** build/vet/test-race green; lint green; a sample agent with a
  read-only MCP compiles to a valid Kiro JSON containing only that agent's server.

## Phase P3 — Research → Skill Cards (§A)

- [ ] **T3.1** Author the **language-agnostic craft** cards under `skills/` with
  `name`/`description`/`source`, tightly scoped, 1–10K tokens, explicit "use
  when":
  `deep-modules`, `diagram-notation`, `reviewing-changes`, `refactoring-safely`,
  `testing-discipline`, `root-cause-analysis`, `reading-unfamiliar-code`,
  `decision-records`, `writing-clear-docs`, `threat-modeling`, `devops-flow`,
  `estimation-and-scope`. — AC-A.1/A.2/A.4 — *M*
- [ ] **T3.2** Grounding check: each card's claims verified against the
  `moonbase-research-books` KB (retrieve + confirm); record source chunk refs in
  the PR. — AC-A.1 — *S*
- [ ] **T3.3** Prohibit raw-PDF injection: add a lint/CI check that no agent/skill
  file embeds >N lines of verbatim book text or a PDF path. — AC-A.3 — *S*
- [ ] **P3 gate:** cards discoverable in the (unscoped) catalogue; lint green;
  no raw-PDF content.

## Phase P4 — Scope the Operatives + Retrieval (§B/§C/§D)

- [ ] **T4.1** Add `skills:` allowlists to the 14 agents per the design matrix
  (soft scope), using only the language-agnostic craft set (AC-A.4). — AC-B.1 — *M*
- [ ] **T4.2** Attach the **language-agnostic reference-server** MCP palette per
  the per-operative matrix (`filesystem` scoped to workspace, `git` RO, `fetch`,
  `memory`, `sequential-thinking`, `time`) with least-privilege + justification;
  platform servers (GitHub, web-search, Playwright) left opt-in. — AC-C.2/C.4/C.6 — *M*
- [ ] **T4.3** Define the single shared read-only `research-kb` retrieval server;
  attach L3 to N0/N4/N274 (explicit, not global). Confirm compiled config matches
  Kiro `mcp.json` shape (AC-C.7). — AC-C.3/C.7/D.2 — *S*
- [ ] **T4.4** Doctrine: add "MCP least-privilege + separation of duties" section
  to the security doctrine; add the L0–L3 placement rule to steering. — AC-C.4/D.1 — *S*
- [ ] **T4.5** ADR `decisions-adr.md`: layered knowledge model + "skills for
  knowledge, MCP for live capability" boundary (Context/Decision/Consequences/
  Alternatives/Reversibility). Update `docs/agent-format.md`. — AC-E.4 — *S*
- [ ] **P4 gate:** `moonbase lint` green for all agents; governance matrix in the
  ADR matches the agent files; scoped catalogues verified for 3 sample agents;
  `moonbase status` lists this spec.

---

## Definition of Done (whole spec)
1. All ACs A–E satisfied with evidence (tests + lint output cited).
2. `go build ./... && go vet ./... && go test -race ./...` green;
   `moonbase lint` green for all 14 agents.
3. New fields are OPTIONAL; an agent with none behaves exactly as pre-spec
   (AC-E.1 regression check).
4. 5 book cards exist, each `source`-cited and grounded (AC-A.2).
5. Governance matrix recorded; no secrets in agent files; MCP least-privilege
   enforced by lint (AC-C.*).
6. ADR + agent-format doc updated (AC-E.4).

## Traceability

| Task | AC | Design § |
|------|----|----------|
| T1.1–1.4 | B.1–B.3, E.3 | §B |
| T2.1–2.4 | C.1/C.2/C.5, E.2/E.3 | §C |
| T3.1–3.3 | A.1–A.3 | §A |
| T4.1–4.5 | B/C/D, E.4 | §B/§C/§D + ADR |

## Non-Goals (restate)
Skill *loading* engine (progressive-skills owns it); fine-tuning; skill
marketplace/remote fetch; building MCP server backends; pipeline/risk-gate changes.

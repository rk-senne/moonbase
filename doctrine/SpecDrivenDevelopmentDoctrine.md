# Spec-Driven Development Doctrine

How the KND Council specifies any software project from zero to production — a reusable method both the operatives and the human commander follow. Load this doctrine before specifying a new project. It is the method, not any single project.

> **The core idea:** You don't write code first. You produce a set of living documents that make the *what*, *why*, *how*, *in-what-order*, and *to-what-standard* unambiguous — then execution (human or agent) becomes mechanical. The spec is the product until the code exists.

---

## Part 1 — The Ten Principles

These are the beliefs the whole method rests on. When a decision is unclear, return to these.

1. **Spec before code.** Every project starts as documents, not files. The spec is the highest-leverage artifact — a day spent specifying saves a week of rework.

2. **Interfaces first, implementations later.** Anything that differs across environments (auth, storage, cache, jobs) hides behind an interface with a fast dev implementation. The real (LDAP/Postgres/Redis/etc.) implementation is a later swap, not a rewrite. This keeps the dev loop instant and makes production hardening cheap.

3. **Ship increments, not the whole thing.** Slice into releases where each is independently demoable and valuable. Ship the first, learn, then decide the rest. Never build everything before showing anyone.

4. **Cheap-now / expensive-later goes early.** If a decision or seam is cheap to establish now but painful to retrofit (schemas, API conventions, interfaces, privacy shape), pull it into the foundation phase — even if the heavy work comes later.

5. **Decisions are recorded, not remembered.** Every significant choice becomes an ADR (Context → Decision → Consequences → Alternatives → Reversibility). Chat messages get lost; ADRs don't. Assumptions are logged explicitly, never presented as facts.

6. **Single source of truth.** One home for the spec. No duplicate copies to drift. If a mirror is needed, automate the sync (a hook), never sync by hand.

7. **Test ships with the code, in the same unit of work.** Never a "we'll add tests later" follow-up. Later never comes. Property-based tests for anything algorithmic.

8. **Every requirement is addressable and traceable.** Acceptance criteria have stable IDs. Code references them. Verification reports pass/fail per ID. Requirement → code → test is one unbroken chain.

9. **Plan for the boundary you control.** Design your own system rigorously; treat external systems (third-party services, other teams, infra you don't own) as integration edges behind interfaces, not things to design.

10. **Governance is part of the build, not an afterthought.** Compliance, security posture, reliability, and process hygiene are planned from the start and gated at production — especially in regulated domains.

---

## Part 2 — The Document Set

A complete spec is a folder of focused documents, each answering one question. Not every project needs all of them, but this is the full menu. Scale to project size.

| Document | Answers | When needed |
|----------|---------|-------------|
| `agent-brief.md` | Where do I start? (mission control) | Always — the entry point |
| `roadmap.md` | When, in what order, what ships? | Always |
| `requirements.md` | What are we building? (numbered ACs) | Always |
| `design.md` | How is it architected? | Always |
| `decisions-adr.md` | Why these choices? (ADRs) | Any project with non-trivial decisions |
| `system-design.md` | Data, flows, state, failure modes | Systems with real data/state |
| `algorithms.md` | What's the hard logic? | Projects with non-trivial algorithms |
| `tasks.md` | The step-by-step, repo-tagged | Always |
| `cross-cutting-layers.md` | The system-wide technical seams | Medium+ projects |
| `production-readiness.md` | What's the gap to prod? | Anything going to production |
| `hardening-plan.md` | How do we get production-ready? | Anything going to production |
| `governance-and-ops.md` | Compliance, security, reliability, process | Regulated / serious projects |

**Small project?** Collapse to: `requirements.md`, `design.md`, `tasks.md`, `decisions-adr.md`. **Large/regulated?** Use all twelve.

**Read order for anyone picking it up cold:** agent-brief → roadmap → requirements → design → (rest as needed).

---

## Part 3 — Layered Thinking

When designing, think in three layers. Missing a layer is how projects surprise you later.

### Layer A — Feature layer
The user-facing capabilities. What the product *does*. This is what everyone thinks of first.

### Layer B — Cross-cutting technical layers
The system-wide concerns that span features. Walk this checklist for every project:
- **Data & schema** — storage, schema design, validation, migrations
- **Caching** — application cache, HTTP cache, invalidation
- **API conventions** — versioning, error envelope, pagination, idempotency, docs
- **Background jobs** — async work, scheduling, queues
- **State management** — where truth lives, client vs server vs ephemeral
- **Config & feature flags** — profiles, validation, runtime toggles
- **Sessions & security config** — cookies, timeouts, CSRF
- **Asset/file handling** — storage, delivery, optimization
- **Frontend asset pipeline** — bundling, cache-busting, compression
- **Testing infrastructure** — unit, property, integration, e2e, contract, coverage gates
- **Observability** — logging, metrics, tracing, error tracking
- **Dev experience** — one-command setup, seed data, hot reload

### Layer C — Governance & operations layer
Especially for regulated/serious projects:
- **Compliance** — data privacy law (POPIA/GDPR/etc.), retention, residency
- **Security posture** — threat model (STRIDE), not just scattered controls
- **Reliability** — backup/DR (RPO/RTO), SLOs, error budgets, alerting
- **Supply chain** — dependency updates, vulnerability scanning, CVE SLA
- **Release management** — semver, changelog, release notes
- **Process** — Definition of Ready/Done, PR templates, CODEOWNERS, incident runbook
- **Cost/FinOps** — what drives cost, right-sizing

**The discipline:** list every layer, mark ✅ covered / 🟡 partial / ❌ not examined. The ❌ items are your real risk.

---

## Part 4 — The Phase / Release Model

Structure delivery as **value increments (releases)**, each containing **milestones**, each containing **missions**.

```
Release (shippable value increment)
  └─ Milestone (coherent chunk of one release)
       └─ Mission (one pipeline run: ~one component + its test, or one backend slice)
            └─ Tasks (the concrete deliverables)
```

**A canonical four-release shape** (adapt to your project):

| Release | Purpose | Exit gate |
|---------|---------|-----------|
| **R1 — Demo-able** | Smallest thing that proves the concept and is usable | First stakeholder demo |
| **R2 — Full product** | The differentiating features | Feature complete |
| **R3 — Production-ready** | Real infra, observability, security, scale, governance | Production go-live gate |
| **R4 — Differentiated** | Best-in-class capabilities | Competitive edge |

**Rules:**
- Each release is independently shippable and demoable.
- Ship R1 before building R2. Let feedback prioritise.
- Foundation work (interfaces, conventions, cheap-now seams) lands in the **first milestone of R1**, even if it serves R3.
- Heavy production work concentrates in R3, as *implementation swaps* behind R1's seams.

---

## Part 5 — Making It Agent-Executable

If AI agents will do the work, the spec needs these properties (they also help humans):

1. **Missions sized to one pipeline run.** Not "build the dashboard" — "build the dashboard-grid component + its test." One reviewable, mergeable unit.

2. **Stable AC-IDs.** `AC-{section}.{index}`. Requirements reference them, code comments cite them, QA reports pass/fail per ID. Full traceability.

3. **Repo + path tags on every mission.** If work spans repos, each mission states its repo. Confirm the agent's write permissions cover it.

4. **Dependency chain explicit.** Which mission blocks which. The critical path to the first demo is spelled out; everything else parallelises around it.

5. **Difficulty + specialist flags.** Mark LARGE/risky missions so oversight/security/devops review triggers.

6. **Definition of Ready + Definition of Done.** DoR: ACs clear, deps met, assumptions confirmed, pattern-to-follow points at real code. DoD: ACs pass by ID, tests ship in-mission, risk classified, handoff complete.

7. **A "Mission 001 — START HERE"** with full detail: objective, ACs, scope in/out, pattern to follow, kickoff. The first mission removes all ambiguity so the pipeline can begin.

8. **Assumption register.** Every assumption logged with risk + who-can-confirm. HIGH-risk ones block their mission until confirmed.

---

## Part 6 — The Step-by-Step Method

How to actually produce all this for a new project, from zero:

**Step 1 — Understand the ground truth.**
Read the existing code/context deeply before proposing anything. If replacing something, study what it does exhaustively (delegate broad exploration to a sub-agent). Never spec against assumptions.

**Step 2 — Draft requirements (the WHAT).**
User stories + acceptance criteria in `WHEN/THEN/SHALL` form. Number every criterion. Cover feature, cross-cutting, and governance layers.

**Step 3 — Resolve the key architecture decisions (the WHY).**
Surface the 5–10 decisions that shape everything (state management, storage, sync model, repo topology, etc.). Discuss options, pick, record each as an ADR with alternatives + reversibility.

**Step 4 — Design the system (the HOW).**
Architecture, component interfaces, data model, data flows, state ownership, failure modes, algorithms. This is `design.md` + `system-design.md` + `algorithms.md`.

**Step 5 — Walk the layers (find the gaps).**
Run Part 3's three-layer checklist. Mark ❌ items. Spec the missing cross-cutting and governance layers. This is where most projects have hidden risk.

**Step 6 — Assess production readiness.**
Honestly list what stands between "works" and "runs in production." Rate by severity. This becomes `production-readiness.md` → `hardening-plan.md`.

**Step 7 — Sequence into releases + missions (the ORDER).**
Build the roadmap: releases → milestones → missions, with effort, dependencies, exit gates, risks. Pull cheap-now foundations into the first milestone.

**Step 8 — Make it executable.**
Write the agent-brief: repo map, AC convention, mission list, DoR/DoD, assumption register, Mission 001. Tag repos, fix agent permissions.

**Step 9 — Review for consistency.**
Read the whole spec as one thing. Hunt contradictions (a decision in one doc that another contradicts), dangling open questions, unaddressed requirements. Resolve every one. Ask the human where genuinely uncertain.

**Step 10 — Keep it alive.**
Single source of truth. Automate backups/mirrors with a hook. Update decisions as ADRs when they change. The spec evolves with the project, not frozen at kickoff.

---

## Part 7 — Discipline Rules (the things that quietly save you)

- **Correct the plan when it's wrong.** Honest feedback beats agreement. If an approach has a flaw, say so.
- **When you catch a decision in chat, write it to the spec.** Unrecorded decisions decay.
- **Turn open questions into deferred decisions with triggers**, not dangling TBDs. "Do X until Y happens" beats "we haven't decided."
- **Resolve inconsistencies the moment you find them** — a spec that contradicts itself is worse than no spec.
- **Estimate in build-time, both human and agent**, and be honest that rework overhead is real (~25%).
- **Flag what you did not verify.** Distinguish confirmed facts from assumptions.
- **Prefer boring, reversible choices** unless the project demands otherwise. Note reversibility on every big decision.

---

## Part 8 — Quick-Start Checklist for a New Project

```
[ ] Read existing code/context (delegate broad scans to a sub-agent)
[ ] requirements.md — numbered ACs across feature/cross-cutting/governance layers
[ ] decisions-adr.md — the 5–10 shaping decisions, each an ADR
[ ] design.md + system-design.md — architecture, data, flows, state, failures
[ ] algorithms.md — if non-trivial logic exists
[ ] Layer walk — ✅/🟡/❌ every cross-cutting + governance layer; spec the ❌s
[ ] cross-cutting-layers.md + governance-and-ops.md — the seams and the gates
[ ] production-readiness.md → hardening-plan.md — the path to prod
[ ] roadmap.md — releases → milestones → missions, effort, deps, gates, risks
[ ] agent-brief.md — repo map, AC convention, mission list, DoR/DoD, Mission 001
[ ] Consistency pass — hunt and resolve contradictions
[ ] Single source of truth + auto-sync hook
[ ] Confirm agent permissions match the repos/paths the missions touch
```

---

*This playbook is the method, not the project. Every project fills it differently — but the shape holds: understand → require → decide → design → find gaps → sequence → make executable → keep alive.*

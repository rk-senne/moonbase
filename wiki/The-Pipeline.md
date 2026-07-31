# The Pipeline

`moonbase mission` runs a **risk-gated pipeline** — a state machine of AI operatives with rework loops.

```
Human Request
    ↓
Numbuh 1  → Requirements (ACs, scope, risks)
    ↓
Numbuh 2  → Design (blueprint, trade-offs, file impact)
    ↓
Numbuh 3  → Implementation (code, tests, build)
    ↓
Numbuh 4  → QA (verify, risk gate)
    ↓                    ↑
    ├── MEDIUM → fix ────┘ (max 2 rework loops)
    ├── HIGH → redesign (back to Numbuh 2)
    ├── CRITICAL → STOP (escalate to human)
    └── LOW ↓
Numbuh 5  → Review (final gate, PR package)
    ↓
Human Approval
```

## The risk gate

Numbuh 4 (QA) classifies every implementation and routes accordingly. **The risk gate is the pipeline's safety core — it always runs and is never bypassed.**

| Risk | Routing |
|------|---------|
| **LOW** | Proceed to Review |
| **MEDIUM** | Route back to Implementation (rework) |
| **HIGH** | Route back to Design (redesign) |
| **CRITICAL** | Stop the pipeline, escalate to a human |

Rework is bounded by `MaxRework` (default 2); exceeding it escalates to a human.

## Adaptive depth (default)

By default, moonbase auto-classifies task complexity and runs the **minimum viable depth**, escalating mid-run if QA signals the shallow path was insufficient — the "start simple, escalate at a ceiling" best practice.

| Depth | Phases | When |
|-------|--------|------|
| `trivial` | 3 → 4 | Short tasks, no complexity signals (typo, rename) |
| `simple` | 1 → 3 → 4 | Moderate tasks (analysis provides enough context) |
| `complex` | 1 → 2 → 3 → 4 → 5 | Long tasks, many signals, multi-scope |

**Mid-pipeline escalation:** if a `trivial`/`simple` pipeline hits MEDIUM/HIGH risk at QA, it un-skips Analysis/Architecture and re-runs — so nothing ships under-reviewed. **QA (Phase 4) always runs at every depth; CRITICAL always stops.** Override with `--fast`, `--full`, or `--depth`.

## Conditional specialists

After QA returns LOW, specialists deploy based on content signals in the pipeline output:

| Operative | Triggers when |
|-----------|--------------|
| Numbuh 0 | >5 files changed, core logic modified, new patterns |
| Numbuh 274 | Auth/secrets touched, new endpoints, dependency CVEs |
| Numbuh 362 | CI/CD, Docker, env vars, deployment config changed |
| Numbuh 9 | Version upgrades, breaking changes, migrations |
| Numbuh 13 | Edge-case coverage needed, fragile flows, parsers |
| Numbuh 86 | Dead code discovered, unused deps, zombie features |
| Numbuh 999 | README needed, ADRs, changelogs |
| Sector Z | Old/mysterious/legacy code touched |

**Parallel fan-out:** independent (read-only) specialists execute concurrently after QA, bounded by `max_specialist_concurrency` (default 4). One failure doesn't cancel siblings; outputs merge deterministically by phase number. Disable with `parallel_specialists: false` or `--sequential`.

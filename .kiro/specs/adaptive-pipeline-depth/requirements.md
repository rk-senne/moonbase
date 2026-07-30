# Requirements: Adaptive Pipeline Depth

## Overview

The KND Council pipeline currently has two modes: full (5 mandatory phases + conditionals) and `--fast` (implementation + QA only). This is a blunt instrument — most tasks don't need the full council, but the user has to manually decide. The result: users either waste cycles running full analysis on trivial tasks, or skip analysis on tasks that needed it.

Adaptive pipeline depth automatically classifies task complexity and selects the minimum viable pipeline depth, escalating mid-run if the risk gate signals the shallow path was insufficient. This follows the Anthropic/OpenAI best practice: **start with a single agent, escalate only when you hit a ceiling**.

The core invariant: **the risk gate is never weakened**. Adaptive depth controls which phases *precede* QA — it never bypasses QA itself.

---

## User Stories

### US-1: Automatic Depth Selection
As a developer running `moonbase mission`, I want the system to automatically choose the right pipeline depth for my task so that trivial fixes don't waste 5 agent calls and complex features don't get under-analyzed.

### US-2: Mid-Pipeline Escalation
As a developer whose "trivial" fix turned out to touch auth, I want the pipeline to automatically promote to a deeper mode when QA flags risks, so that nothing ships under-reviewed.

### US-3: Explicit Override
As a developer who knows exactly what depth I want, I want `--fast`, `--full`, and `--depth` flags to override auto-classification, so that I retain full control.

### US-4: Flywheel Visibility
As a developer reviewing pipeline efficiency, I want to see which depth was chosen, whether escalation occurred, and why — in the flywheel log — so that I can tune the system over time.

### US-5: Backward Compatibility
As a developer using `--fast` today, I want my existing workflows to keep working identically, so that adaptive depth is purely additive.

---

## Acceptance Criteria

### AC-1: Task Complexity Classification

#### AC-1.1: Trivial Task Detection
- **WHEN** a mission task is ≤ 80 characters AND contains no complexity signals
- **THEN** it is classified as `trivial`
- **SHALL** match tasks like "fix typo in README", "rename getUserName to getUsername", "remove unused import"

#### AC-1.2: Complex Task Detection
- **WHEN** a mission task contains complexity keywords (e.g., "add", "implement", "redesign", "migrate", "refactor architecture", "new endpoint", "rate limit", "pagination") OR exceeds 200 characters OR references ≥ 3 distinct file paths/packages
- **THEN** it is classified as `complex`
- **SHALL** match tasks like "add rate limiting to the API with per-user quotas and Redis backing"

#### AC-1.3: Simple Task Detection (Default)
- **WHEN** a task does not meet trivial or complex criteria
- **THEN** it is classified as `simple`
- **SHALL** be the fallback — ambiguity resolves to `simple`, not `trivial`

#### AC-1.4: Classification Reuses Reasoning-Protocol Ladder
- **WHEN** classifying task complexity
- **THEN** the three tiers map directly to the reasoning-protocol's task-scaling table: trivial → "Fix directly, verify builds"; simple → "Read context → implement → test"; complex → "Full protocol"
- **SHALL** use the same signal vocabulary documented in `.kiro/steering/reasoning-protocol.md`

---

### AC-2: Depth Selection

#### AC-2.1: Trivial → Implement + QA
- **WHEN** task is classified `trivial`
- **THEN** pipeline runs Phase 3 (Implementation) → Phase 4 (QA) only
- **SHALL** skip Phases 1, 2, 5 (Analysis, Architecture, Review)
- **SHALL** still run Phase 5 (Review) if QA risk is anything other than LOW

#### AC-2.2: Simple → Analysis + Implement + QA
- **WHEN** task is classified `simple`
- **THEN** pipeline runs Phase 1 (Analysis) → Phase 3 (Implementation) → Phase 4 (QA)
- **SHALL** skip Phase 2 (Architecture) — analysis provides enough context
- **SHALL** proceed to Phase 5 (Review) after QA passes with LOW risk

#### AC-2.3: Complex → Full Council
- **WHEN** task is classified `complex`
- **THEN** pipeline runs all 5 mandatory phases: Analysis → Architecture → Implementation → QA → Review
- **SHALL** behave identically to the current default `moonbase mission` behavior

#### AC-2.4: Conditional Phases Unaffected
- **WHEN** any depth is selected
- **THEN** conditional phases (6, 7, 8) still evaluate their triggers as normal
- **SHALL NOT** suppress conditional specialists based on depth selection

---

### AC-3: Mid-Pipeline Escalation

#### AC-3.1: QA-Triggered Escalation (Trivial → Simple)
- **WHEN** a `trivial` pipeline reaches QA (Phase 4) AND QA returns MEDIUM risk
- **THEN** the pipeline escalates to `simple` depth, adding Phase 1 (Analysis) before reworking Phase 3
- **SHALL** route: QA (MEDIUM) → add Phase 1 → re-run Phase 3 → re-run Phase 4

#### AC-3.2: QA-Triggered Escalation (Trivial/Simple → Complex)
- **WHEN** a `trivial` or `simple` pipeline reaches QA AND QA returns HIGH risk
- **THEN** the pipeline escalates to `complex` depth, adding Phase 2 (Architecture) before reworking
- **SHALL** route: QA (HIGH) → add Phase 2 → re-run Phase 3 → re-run Phase 4
- **SHALL** add Phase 1 (Analysis) first if not already run

#### AC-3.3: Escalation Respects MaxRework
- **WHEN** escalation triggers a rework loop
- **THEN** the existing `MaxRework` limit (default 2) still applies
- **SHALL** escalate to human (CRITICAL stop) if max rework is exceeded post-escalation

#### AC-3.4: CRITICAL Risk Always Stops
- **WHEN** QA returns CRITICAL risk at any depth
- **THEN** the pipeline stops and escalates to human — no automatic escalation can override this
- **SHALL** maintain the existing `p.Stop("CRITICAL risk — pipeline stopped")` behavior unchanged

#### AC-3.5: Risk Gate Never Weakened
- **WHEN** adaptive depth is active
- **THEN** the risk gate logic in `ParseRiskGate` and `ApplyRiskGate` remains unmodified
- **SHALL NOT** change the thresholds, parsing, or routing decisions of the risk gate
- **SHALL NOT** allow a shallow pipeline to bypass QA under any circumstances

---

### AC-4: CLI Flags

#### AC-4.1: --fast Preserved
- **WHEN** `moonbase mission --fast "task"` is used
- **THEN** behavior is identical to current: Phase 3 + Phase 4 only, informational risk gate
- **SHALL NOT** trigger auto-classification or mid-pipeline escalation
- **SHALL** log depth as `"override:fast"` in flywheel

#### AC-4.2: --full Flag Added
- **WHEN** `moonbase mission --full "task"` is used
- **THEN** all 5 mandatory phases run regardless of task complexity
- **SHALL** log depth as `"override:full"` in flywheel
- **SHALL** behave identically to current `moonbase mission` (no `--fast`)

#### AC-4.3: --depth Flag Added
- **WHEN** `moonbase mission --depth trivial|simple|complex "task"` is used
- **THEN** the specified depth is used without auto-classification
- **SHALL** still allow mid-pipeline escalation (override selects starting depth, not ceiling)
- **SHALL** log depth as `"override:<value>"` in flywheel

#### AC-4.4: Flag Mutual Exclusivity
- **WHEN** more than one of `--fast`, `--full`, or `--depth` is specified
- **THEN** the CLI rejects the command with a clear error message
- **SHALL** print: "❌ Flags --fast, --full, and --depth are mutually exclusive."

#### AC-4.5: Default Behavior Change
- **WHEN** `moonbase mission "task"` is used with no depth flags
- **THEN** auto-classification determines depth (this is the new default)
- **SHALL** be the only behavioral change from today's default (which always runs full)

---

### AC-5: Flywheel Logging

#### AC-5.1: Depth Logged on Every Entry
- **WHEN** a flywheel entry is written for any phase
- **THEN** it includes a `"depth"` field with the pipeline's effective depth
- **SHALL** use values: `"trivial"`, `"simple"`, `"complex"`, `"override:fast"`, `"override:full"`, `"override:<value>"`

#### AC-5.2: Classification Reason Logged
- **WHEN** auto-classification runs (no override flag)
- **THEN** the first flywheel entry for that trace includes a `"depth_reason"` field
- **SHALL** contain a brief explanation: e.g., `"short task, no complexity signals"`, `"contains 'implement' + 'endpoint' keywords"`, `"task > 200 chars with multi-package scope"`

#### AC-5.3: Escalation Logged
- **WHEN** mid-pipeline escalation occurs
- **THEN** a flywheel entry is written with `"outcome": "escalated"` and `"escalated_from"` / `"escalated_to"` fields
- **SHALL** include the risk level that triggered escalation

#### AC-5.4: Schema Version Unchanged
- **WHEN** new fields are added to FlywheelEntry
- **THEN** the schema version remains `1` (new optional fields don't require a bump per the flywheel evolution contract)
- **SHALL** use `omitempty` JSON tags on all new fields

---

### AC-6: Observability

#### AC-6.1: Depth Announced at Start
- **WHEN** a mission starts with auto-classification
- **THEN** the CLI prints the chosen depth and reason before phase execution
- **SHALL** format: `"   Depth: <trivial|simple|complex> (<reason>)"`

#### AC-6.2: Escalation Announced
- **WHEN** mid-pipeline escalation occurs
- **THEN** the CLI prints a clear message before adding phases
- **SHALL** format: `"   ⬆️  Escalating: <from> → <to> (QA risk: <level>)"`

---

## Scope

### In Scope
- Task complexity classifier (heuristic, not AI-based — fast, deterministic, testable)
- Depth selection logic mapping complexity → phases
- Mid-pipeline escalation when risk gate signals insufficient depth
- `--full` and `--depth` CLI flags
- Flywheel field additions for depth tracking
- CLI output for depth/escalation visibility

### Out of Scope
- AI-based task classification (too slow, non-deterministic — a future enhancement)
- Changing the risk gate logic or thresholds
- Changing conditional specialist trigger logic
- Changing `--fast` behavior (it remains a hard override with no escalation)
- Per-agent depth awareness (agents don't know about depth — they just execute)
- Configuration file for classification thresholds (use constants first; extract to config if flywheel data shows tuning is needed)

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Classifier is too aggressive (marks complex tasks as trivial) | Medium | High | Default ambiguity to `simple`; escalation catches mistakes; log reasons for flywheel tuning |
| Escalation creates confusing output (phases appear mid-run) | Low | Medium | Clear `⬆️ Escalating` message; TUI shows added phases |
| `--fast` users confused by new default behavior | Low | Low | `--fast` behavior unchanged; only default (no flags) changes |
| New flywheel fields break existing readers | Low | Low | `omitempty` tags; schema version unchanged per evolution contract |
| Classification adds latency to mission start | Very Low | Low | Pure string analysis — microseconds, not AI calls |

---

## Dependencies

| Dependency | Version | Purpose | Status |
|------------|---------|---------|--------|
| None | — | Classification is pure Go string logic | No new deps |

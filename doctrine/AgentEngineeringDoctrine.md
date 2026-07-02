# Agent Engineering Doctrine

How Moonbase agents are built, informed by industry best practices. Distilled from Anthropic/OpenAI agent architecture guides, Karpathy's autoresearch methodology, and the "smallest useful loop" principle.

---

## Part 1 — The Agent Loop (Foundation)

Every agent — no matter how complex — is built on the same fundamental loop:

```
read input → decide → act → observe → repeat
```

This is the skeleton. Moonbase agents implement this as:

```
Receive task
    ↓
Read context (specs, code, history)
    ↓
Decide (what needs doing, what tools to use, whether to ask)
    ↓
Act (write code, run tests, produce output)
    ↓
Observe (did it work? what's the risk? what changed?)
    ↓
Hand off or repeat
```

**The loop is the agent.** Everything else — personality, tools, routing — is configuration of what happens inside each step of the loop.

---

## Part 2 — The Three Components (from OpenAI Practical Guide)

Every agent has exactly three core components:

### 1. Model (the reasoning engine)
The LLM that powers decisions. In moonbase, this is whichever backend you deploy to (kiro-cli → Claude/GPT/etc).

### 2. Tools (the action surface)
External functions the agent can call. In moonbase, these are defined in frontmatter:
- `read`, `write`, `shell`, `grep`, `glob`, `code` — file/code operations
- `knowledge`, `web_search`, `web_fetch` — information gathering
- `subagent` — delegating to other agents
- `use_aws` — cloud operations

### 3. Instructions (the behaviour envelope)
The prompt that defines how the agent behaves. In moonbase, this is the full markdown body of the agent file — identity, purpose, output formats, behaviour rules, boundaries.

**Key insight:** Tools expand what the agent *can* do. Instructions constrain what the agent *should* do. The model decides what to *actually* do.

---

## Part 3 — Orchestration Patterns

### Single-Agent (default for most tasks)
One agent with the right tools handles the work end-to-end. Prefer this until it breaks down.

**When to use:** Task fits one role. Clear scope. Less than ~15 tools needed.

### Multi-Agent Pipeline (Moonbase KND Council)
Sequential handoff: each agent completes its phase, passes context to the next.

```
Analyst → Architect → Implementer → QA → Reviewer
```

**When to use:** Complex workflows where each phase needs different tools, different constraints, and different verification.

### Manager Pattern (agent-as-tool)
A central agent coordinates specialists, calling them as tools and synthesizing results.

**When to use:** When you need one agent to maintain the conversation thread while delegating subtasks.

### Decentralized (agent-to-agent handoff)
Agents hand off to each other based on content routing. No central coordinator.

**When to use:** Triage scenarios. The right specialist depends on what the user says.

### Moonbase Implementation

Moonbase uses the **pipeline pattern** for structured development work (KND Council) and the **decentralized pattern** for specialist routing (Numbuh 4 routes to Numbuh 274 for security, Numbuh 2 routes to Sector Z for legacy context).

The `routing` field in agent frontmatter defines which agents can be reached:
```yaml
routing:
  available: [numbuh-2, numbuh-274, sector-z]  # can hand off to
  trusted: [numbuh-2]                          # can delegate without approval
```

---

## Part 4 — The Guardrails System

Guardrails are layered defenses that keep agents operating safely. They are NOT optional — they are what separates a useful agent from a dangerous one.

### Types of Guardrails in Moonbase

| Type | Implementation |
|------|---------------|
| **Tool boundaries** | Frontmatter `tools` + `shell.allowed_commands` + `write.denied` |
| **Evidence requirement** | Operating Protocol: "Support every claim with evidence" |
| **Human intervention** | Stop Conditions: escalate on secrets, destructive actions, HIGH risk |
| **Scope discipline** | Behaviour Rules: "Do not modify files outside scope" |
| **Risk classification** | QA risk gate: LOW/MEDIUM/HIGH/CRITICAL routing |
| **Uncertainty protocol** | Human Interaction: CERTAIN/LIKELY/UNCERTAIN/UNKNOWN |

### The Guardrails Heuristic (from OpenAI guide)

1. Focus on data privacy and content safety first
2. Add new guardrails based on real-world edge cases and failures
3. Optimize for both security and user experience

### Human Intervention Triggers

Two conditions that ALWAYS warrant human involvement:
1. **Exceeding failure thresholds** — max rework loops hit, repeated failures
2. **High-risk actions** — irreversible, sensitive, or high-stakes operations

---

## Part 5 — The Autoresearch Principle (from Karpathy)

The key insight from autoresearch: **an agent with a clear metric, a fixed budget, and constrained scope can make real progress autonomously.**

### How this applies to Moonbase

When deploying agents for development work, structure the environment so:

1. **The metric is clear** — Acceptance Criteria with stable IDs. Pass/fail per AC.
2. **The budget is fixed** — Scoped to one mission. One reviewable unit of work.
3. **The scope is constrained** — Agent can only modify files within its permissions.
4. **The evaluation is honest** — Tests run by a different agent (QA ≠ implementer).
5. **The loop is autonomous** — Agent keeps working until it succeeds or hits a stop condition.

### The "Never Stop" Principle

autoresearch tells the agent: "Do not ask to continue. Do not pause. Keep running experiments until interrupted."

In Moonbase, this translates to: agents should be **biased toward action**, not toward asking permission. The questioning protocol (CERTAIN/LIKELY/UNCERTAIN/UNKNOWN) is the guard — but for CERTAIN and LIKELY decisions, the agent proceeds without asking.

### The "Controlled Experiment" Structure

autoresearch locks down:
- The data (validation set — can't be changed)
- The metric (val_bpb — can't be gamed)
- The budget (5 minutes — can't be exceeded)

In Moonbase, the equivalent controls are:
- The spec (requirements.md — defines what success looks like)
- The tests (QA agent verifies against ACs — not the implementer)
- The scope (agent frontmatter — defines what can be touched)

---

## Part 6 — Tool Design Principles (from OpenAI guide)

### Three types of tools

1. **Data retrieval** — read files, search knowledge, query APIs
2. **Action execution** — write code, run commands, deploy
3. **Orchestration** — hand off to another agent, spawn sub-tasks

### Best practices for tool definitions

- **Clear names** — `get_weather` not `tool_1`
- **Descriptive parameters** — tell the model what format is expected
- **Single responsibility** — one tool does one thing
- **Error surfaces** — tools return structured errors, not raw exceptions
- **Scoped permissions** — tools only access what's needed

### Moonbase tool governance

Each agent's tools are explicitly declared in frontmatter. The `auto_tools` list defines what runs without human approval. Everything else may require confirmation.

```yaml
tools: [read, write, shell, grep, glob, code]  # available
auto_tools: [read, grep, glob, code]           # auto-execute
shell:
  allowed_commands: ["go test", "npm test"]    # constrained
  read_only: false
write:
  auto: ["src/**", "tests/**"]                 # where it can write
  denied: ["config/**", ".env"]                # where it can't
```

---

## Part 7 — The Exit Condition

From the Substack article: "Every orchestration approach needs a concept of a 'run', implemented as a loop that lets agents operate until an exit condition is reached."

### Exit conditions in Moonbase

| Condition | What happens |
|-----------|-------------|
| Task complete (all ACs satisfied) | Agent produces final output, hands off |
| Stop condition triggered | Agent halts, escalates to human |
| Risk gate fires | Pipeline reroutes or stops |
| Max rework exceeded | Pipeline escalates to human |
| Agent explicitly says DONE | Pipeline advances |
| Human aborts | Everything stops immediately |

### The "no vague endings" rule

Every agent response must either:
1. Move the pipeline forward (handoff to next agent)
2. Route back for rework (with specific feedback)
3. Escalate (with evidence)
4. Block (with clear reason)
5. Complete (with proof)

No agent may end ambiguously. The loop must always resolve.

---

## Part 8 — Applying These Principles

### When building a new agent

1. Start with the loop: what does this agent read → decide → do → observe?
2. Define the three components: what model, what tools, what instructions?
3. Set guardrails: what can't this agent do? When does it stop?
4. Define the exit condition: how does the loop end?
5. Connect to the pipeline: who hands off to this agent, and who does it hand off to?

### When improving an existing agent

1. Is the agent's loop clear? Can you trace read → decide → act → observe?
2. Are guardrails proportional? Too loose = dangerous. Too tight = useless.
3. Is the exit condition unambiguous?
4. Does the agent ask the right questions at the right time?
5. Can the agent recover from failure without human help?

### When evaluating agent quality

The three questions from the Substack article that matter:
1. Where is the model call? (What's actually doing the reasoning?)
2. Who executes the tools? (What's taking action in the real world?)
3. How does the loop stop? (What prevents infinite execution?)

---

## Summary

| Principle | Source | Moonbase Implementation |
|-----------|--------|------------------------|
| Agent = loop around LLM | Substack | Pipeline phases, each running decide→act→observe |
| Three components (model, tools, instructions) | OpenAI Guide | YAML frontmatter (tools) + markdown body (instructions) |
| Guardrails are layered | OpenAI Guide | Tool boundaries + evidence rule + stop conditions + risk gate |
| Human intervention for high-risk | OpenAI Guide | Stop conditions + UNCERTAIN/UNKNOWN questioning |
| Clear metric + fixed budget = autonomous progress | autoresearch | AC-IDs + scoped missions + separate QA |
| Never stop (bias toward action) | autoresearch | CERTAIN/LIKELY = proceed without asking |
| Exit conditions are explicit | All three | Handoff protocol, stop conditions, risk gate |
| Start simple, add complexity only when needed | All three | Single agent first, multi-agent only when tool count exceeds clarity |

---

*The best agent systems are the ones where, if you remove all the personality and style, you can still clearly see: the loop, the tools, the guardrails, and the exit.*

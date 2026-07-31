# 🌙 Moonbase Wiki

**K.N.D. Tactical Operations Terminal** — a 14-agent AI development pipeline with spec-driven methodology, built for Kiro CLI and compatible with any AI backend.

## What is Moonbase?

Moonbase is an opinionated, backend-agnostic **multi-agent orchestration framework for spec-driven software development** — a single self-contained Go binary that runs a "council" of 14 markdown-defined AI operatives through a risk-gated pipeline (Requirements → Design → Implementation → QA → Review, with conditional specialists).

```bash
moonbase init                                    # make any project agent-ready
moonbase deploy 4 "check auth"                   # deploy Numbuh 4 with a task
moonbase mission "add pagination to /users API"  # run the full pipeline
```

## Quick links

- **[[Installation]]** — install script, go install, release download, source
- **[[CLI Reference|CLI-Reference]]** — every command and flag
- **[[The Pipeline|The-Pipeline]]** — phases, risk gate, adaptive depth, parallel specialists
- **[[Agents]]** — the 14 operatives and the agent file format
- **[[Architecture]]** — packages, data flow, TUI
- **[[Configuration]]** — config.yaml options
- **[[Kiro Native Interop|Kiro-Native-Interop]]** — MCP servers + moonbase compile
- **[[Skills & Prompts|Skills-and-Prompts]]** — progressive skill loading
- **[[Flywheel & Observability|Flywheel-and-Observability]]** — token/cost + learning insights
- **[[Contributing]]** — dev workflow and quality gates

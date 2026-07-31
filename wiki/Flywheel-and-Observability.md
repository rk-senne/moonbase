# Flywheel & Observability

Moonbase logs pipeline execution data to `~/.moonbase/flywheel.jsonl` (one JSON record per phase). Over time, patterns emerge — which agents struggle, which phases get reworked, where the pipeline bottlenecks.

```bash
moonbase flywheel   # show learning insights + token/cost summary
```

## Learning insights

The flywheel surfaces: total missions, rework rate, risk-level distribution, longest phase, and the adaptive **depth** chosen per run (plus any mid-pipeline escalation `escalated_from → escalated_to`).

## Token & cost

When a mission uses an API backend that reports usage (**OpenAI, Anthropic, Kimi**), moonbase captures per-phase prompt/completion tokens and estimates cost from a configurable price table:

```
💰 Token & Cost Summary:
   Total tokens:     275K prompt / 70K completion
   Total est. cost:  $1.39
💰 Cost per Agent (avg per invocation):
   numbuh-3   $0.32  (61K prompt, 16K completion)
   …
💰 Cost per Mission:  avg $0.69 · most expensive 20260730T… ($0.87)
💰 Cost-Heavy Phase:  Phase 3 (numbuh-3) — 46% of total cost
```

> **Note:** the default `kiro-cli` backend runs as a subprocess and does not expose token counts, so it reports no usage — the cost section shows "(no usage data)" for those runs. The infrastructure future-proofs the rest.

## Token budget

Cap runaway missions (opt-in; disabled by default):

```yaml
token_budget:
  max_tokens_per_mission: 500000   # hard cap (0 = unlimited)
  warn_threshold_pct: 80           # warn at 80% of budget
model_pricing:
  gpt-4o: { prompt: 2.50, completion: 10.00 }
```

The pipeline warns at the threshold and hard-stops (outcome `budget_exceeded`, with a checkpoint saved) if the cap is exceeded.

## Schema

The flywheel record uses a versioned schema (`v: 1`). New fields (tokens, cost, depth, parallel group) are added as optional (`omitempty`) — old records and readers remain valid.

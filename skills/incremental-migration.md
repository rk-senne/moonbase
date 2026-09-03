---
name: incremental-migration
description: Strangler fig, branch by abstraction, parallel run, tracer write, and safe data splitting. From Monolith to Microservices (Newman).
---

# Incremental Migration

## Deployment Is Not Release

Deploying to production does not mean traffic flows. Deploy the new path early — even
returning `501 Not Implemented` — to de-risk the pipeline, then release separately by
redirecting calls. Strangler fig, parallel run, and canary all depend on this split.

## Choosing a Pattern

- **Strangler fig** — redirect at the perimeter (usually a proxy): identify the slice,
  implement it in the new home, reroute calls.
- **Branch by abstraction** — when the target is buried: create an abstraction, move
  callers onto it with no behavior change, add the new implementation alongside the
  old, switch, then delete the old path. Beats a long-lived branch.
- **Decorating collaborator** — when the old code can't change: let its call proceed
  and trigger new behavior off the result.
- **Change data capture** — when neither call path nor code can change. Transaction-log
  polling is cleanest; use database triggers very sparingly.

## Verify Before You Trust

**Parallel run:** call both implementations and compare, keeping the old as source of
truth. Compare latency and timeouts too, not just values. Expensive — reserve for real
risk. Canary exposes a subset of traffic; dark launching exercises it invisibly. Set
thresholds (95th-percentile latency, error rate) and roll back automatically.

## Freeze Behavior Mid-Flight

Fixing bugs or adding features in the new implementation before the migration finishes
means rollback reintroduces old bugs or removes shipped features. Keep each migration
small — the longer it runs, the greater the pressure to slip changes in.

## Moving Data

- **Tracer write** — tolerate two sources of truth briefly: write to both, migrate one
  slice at a time, move readers, retire the old.
- Splitting the schema first surfaces performance and integrity problems early but
  delivers little short-term value. Splitting code first is faster — the trap is
  stopping there and keeping a shared database forever.

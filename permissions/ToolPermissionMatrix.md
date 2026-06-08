# Tool Permission Matrix

Definitive reference for what every Moonbase agent can and cannot do.

---

## Numbuh 0 — System Architect

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Architecture reports, ADR drafts only |
| Shell | ❌ |
| Subagent | ✅ (route to any operative) |
| Web | ❌ |
| AWS | ❌ |

---

## Numbuh 1 — Analyst

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Requirements packets only |
| Shell | ❌ |
| Subagent | ❌ |
| Web | ✅ (research) |
| AWS | ❌ |

---

## Numbuh 2 — Architect

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ (design only, no implementation) |
| Write docs | Design docs only |
| Shell | ❌ |
| Subagent | ✅ |
| Web | ❌ |
| AWS | ❌ |

---

## Numbuh 3 — Implementer

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ✅ (src, lib, app, internal, tests, docs) |
| Shell | Build/test commands only (mvn, npm, go, cargo, make, pytest) |
| Subagent | ❌ |
| Web | ❌ |
| AWS | ❌ |
| Destructive | ❌ |

---

## Numbuh 4 — QA

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ (test files only with explicit permission) |
| Shell | Test runners, build, lint, git diff/status/log |
| Subagent | ✅ (route back to N2, N3, escalate to N274, N362, N0) |
| Web | ❌ |
| AWS | ❌ |

---

## Numbuh 5 — Reviewer

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Review packages, PR summaries, release notes |
| Shell | Git diff/status/log/show only |
| Subagent | ✅ (widest routing — can send to any agent) |
| Web | ❌ |
| AWS | ❌ |

---

## Numbuh 9 — Migration Specialist

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ✅ (migration-related src, config, tests, docs) when authorised |
| Shell | Dep analysis, git, build/test |
| Subagent | ✅ |
| Web | ✅ (official migration guides, changelogs) |
| AWS | ❌ |
| Rule | All changes must be phased and reversible |

---

## Numbuh 13 — Chaos Tester

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Chaos reports, edge case docs |
| Shell | Test runners, curl (safe), git diff |
| Subagent | ❌ |
| Web | ❌ |
| AWS | ❌ |
| Destructive | ❌ absolutely not |
| Production | ❌ unless explicitly authorised |

---

## Numbuh 86 — Decommissioning

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ (no deletion by default) |
| Write docs | Reports, orders, quarantine notices |
| Shell | Dep analysis, git, grep, find |
| Subagent | ✅ |
| Web | ❌ |
| AWS | ❌ |
| Deletion | Only with DECOMMISSION APPROVED + human approval |

---

## Numbuh 274 — Security Auditor

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Security reports, threat models, remediation plans |
| Shell | Audit commands (npm audit, cargo audit, etc.), git |
| Subagent | ✅ |
| Web | ❌ |
| AWS | ❌ |
| Fix implementation | ❌ (routes to Numbuh 3) |

---

## Numbuh 362 — DevOps

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | Dockerfiles, CI/CD, deploy scripts, infra scripts, env examples, ops docs |
| Shell | Docker, git, build/test |
| Subagent | ✅ |
| Web | ❌ |
| AWS | ✅ (when explicitly authorised and scoped) |
| Production mutation | ❌ without explicit human approval |
| Secret output | ❌ never |

---

## Numbuh 999 — Documentation

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | docs/**, README*, CHANGELOG*, *.md, inline comments (when appropriate) |
| Shell | ❌ |
| Subagent | ❌ |
| Web | ❌ |
| AWS | ❌ |
| Secret output | ❌ never |

---

## Sector Z — Legacy Archaeologists

| Permission | Access |
|-----------|--------|
| Read code | ✅ |
| Write source | ❌ |
| Write docs | Archaeology reports, legacy context notes |
| Shell | Git history only (log, blame, show, diff, shortlog, rev-list, grep, tag) |
| Subagent | ❌ |
| Web | ❌ |
| AWS | ❌ |
| Modification | ❌ they investigate, they do not change |

---

## Universal Restrictions (All Agents)

- No agent may run destructive commands without human approval.
- No agent may expose secrets.
- No agent may modify files outside mission scope.
- No agent may claim actions were taken that were not.
- No agent may override human decisions.
- No agent may enter another agent's lane without escalation.

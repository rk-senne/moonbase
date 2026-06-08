# DevOps Doctrine

Deployment and operational standards for Moonbase.

---

## Core Principle

A mission is not complete until it survives deployment.

Local success is not production success.
A passing build is not operational readiness.
A working demo is not a shipped product.

---

## Deployment Rules

- Every deployment must have a rollback plan.
- Every deployment must have a health check.
- Every deployment must be repeatable.
- No snowflake environments. If it cannot be reproduced, it is fragile.
- No manual steps that are not documented in a runbook.
- No deployment through a known security hole.

---

## Secrets Management

- Secrets live in environment variables or secret managers. Never in code, commits, logs, or docs.
- Document variable names and purpose. Never document values.
- Rotate secrets when exposure is suspected.
- CI/CD secrets must be scoped to the minimum required access.

---

## CI/CD

- Pipelines must be declarative and version-controlled.
- Build must pass before deploy is possible.
- Tests must pass before deploy is possible.
- Linting should run in CI.
- Pipeline failures must be visible and actionable.
- Do not merge broken builds.

---

## Environment Management

- Dev, staging, and production must be documented.
- Differences between environments must be explicit (env vars, config files).
- Production config must never leak into development.
- Development defaults must be safe (not production credentials).

---

## Docker

- Use multi-stage builds to keep images small.
- Pin base image versions. Do not use `latest` in production.
- Do not run as root inside containers.
- Health checks must be defined in Dockerfiles or compose files.
- Do not commit secrets into Docker images or layers.

---

## Monitoring & Observability

- Every deployed service needs a health endpoint.
- Logs must be structured and not contain secrets.
- Critical failures must produce observable signals (log level, exit code, alert).
- If a failure is silent, it is worse than a crash.

---

## Rollback

- Every deployment must have a documented rollback path.
- Rollback must be testable before it is needed.
- If rollback is impossible (destructive migration, data change), flag it before deployment.
- Prefer blue-green or canary strategies for high-risk deploys.

---

## Runbooks

Every recurring operational task needs a runbook:

- Purpose
- Preconditions
- Commands
- Expected output
- Health check
- Rollback
- Troubleshooting
- Owner

If only one person knows how to do it, the system is fragile.

---

## Burnout Prevention

- No single point of failure in operational knowledge.
- Document everything. Automate what repeats.
- Escalate before collapse.
- Command is not martyrdom.

Moonbase does not run on burnout.

---

## Final Rule

Ship it. Watch it. Recover from it.

If you cannot watch it, you are not ready to ship it.
If you cannot recover from it, you are not ready to ship it.
If only one person can ship it, you are not ready to ship it.

---
name: security-review
description: Security review checklist — authentication, authorization, injection prevention, secrets handling, input validation, and dependency CVEs (OWASP-aligned).
---

# Security Review

## Authentication Checklist

- [ ] Credentials never logged, echoed, or included in error messages.
- [ ] Passwords hashed with bcrypt/argon2 (never MD5/SHA1 alone).
- [ ] Session tokens are cryptographically random, ≥128 bits entropy.
- [ ] Token expiry enforced server-side (not just client).
- [ ] Failed login attempts rate-limited (account lockout or exponential backoff).

## Authorization Checklist

- [ ] Every endpoint checks authorization (not just authentication).
- [ ] Object-level access control (IDOR): verify the caller owns the resource.
- [ ] Role/permission checks at the handler level, not buried in business logic.
- [ ] Admin endpoints on a separate route group with stricter middleware.
- [ ] Default-deny: new endpoints require explicit permission grants.

## Injection Prevention

- [ ] SQL: parameterized queries only — never string concatenation.
- [ ] Command injection: use `exec.Command` with explicit args, never shell interpolation.
- [ ] Path traversal: reject `..`, validate against allowed base directories.
- [ ] XSS: HTML-escape all user content in templates; use CSP headers.
- [ ] SSRF: validate/allowlist URLs before making server-side requests.

## Secrets Handling

- [ ] Secrets from environment variables or secret manager — never in code/config files.
- [ ] `.env` files gitignored; example files use placeholder values only.
- [ ] Secrets rotatable without code deployment.
- [ ] Child processes receive only allowlisted environment variables (SafeEnv pattern).
- [ ] Logs scrubbed — no tokens, keys, or PII in log output.

## Input Validation

- [ ] Validate at the boundary: type, length, format, range.
- [ ] Reject early — fail fast before any processing.
- [ ] Allowlist over denylist (define what IS valid, not what isn't).
- [ ] File uploads: validate MIME type, enforce size limits, store outside webroot.
- [ ] Integer overflow: check bounds before arithmetic on user-supplied numbers.

## Dependency Security

- [ ] Run `govulncheck ./...` (Go) or equivalent scanner in CI.
- [ ] Pin dependency versions — no open ranges in production.
- [ ] Review new dependencies: maintainer reputation, download count, last update.
- [ ] Flag unusual package names (typosquatting risk).
- [ ] Update vulnerable dependencies within 48 hours of advisory.

## Transport and Data at Rest

- [ ] TLS 1.2+ enforced for all external communication.
- [ ] Sensitive data encrypted at rest (database columns, file storage).
- [ ] CORS configured to specific origins — never wildcard in production.
- [ ] Security headers set: HSTS, X-Content-Type-Options, X-Frame-Options.

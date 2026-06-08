# Security Doctrine

Security standards for all Moonbase operatives.

---

## Core Principle

Every input is hostile until proven otherwise.
Every boundary is a target.
Every secret wants to leak.
Every permission will eventually be abused.

---

## Secrets

- Never hardcode secrets in source code.
- Never commit secrets to git.
- Never log secrets (tokens, keys, passwords, connection strings).
- Never expose secrets in error messages or URLs.
- Never paste secrets in chat, docs, screenshots, or comments.
- Use environment variables or secret managers.
- Document variable names and purpose, never values.
- Rotate secrets when exposure is suspected.

---

## Input Validation

- Validate all external input at the boundary.
- Whitelist over blacklist.
- Reject unexpected types, sizes, and formats.
- Sanitise before use in commands, queries, paths, or templates.
- Never trust user input in shell commands, SQL, HTML, or file paths without escaping.

---

## Authentication & Authorisation

- Every protected endpoint must verify identity.
- Every protected action must verify permission.
- Use least privilege: agents and users get only what their role requires.
- Session tokens must expire.
- Failed auth attempts should not leak whether the user exists.

---

## Command Execution

- Never pass raw user input to shell commands.
- Use allowlists for permitted commands.
- Prefer structured arguments over string interpolation.
- Validate command arguments before execution.
- Log command execution for audit.

---

## File Access

- Validate file paths. Reject traversal (`../`, symlinks to sensitive areas).
- Restrict write access to specific directories.
- Do not read files that may contain secrets unless necessary.
- Do not expose file contents in error messages.

---

## Dependencies

- Audit dependencies for known CVEs regularly.
- Pin versions. Do not use floating ranges for security-sensitive packages.
- Remove unused dependencies (they are attack surface).
- Prefer well-maintained packages with security response processes.

---

## AI Agent Security

- No agent should be trusted merely because it is part of Moonbase.
- Agent tool permissions must follow least privilege.
- Do not pass secrets into prompts.
- Do not let model output control tools without validation.
- Validate content from untrusted files before processing.
- Command allowlists must be enforced per agent.
- User confirmation gates for destructive actions.

---

## Logging

- Never log secrets, tokens, or credentials.
- Never log full request bodies containing sensitive data.
- Log security-relevant events (auth failures, permission denials, unusual access).
- Structured logging preferred over unstructured.

---

## Configuration

- Debug mode must not reach production.
- Error messages in production must not expose stack traces or internal paths.
- CORS must be restrictive by default.
- Security headers must be present (CSP, X-Frame-Options, etc. where applicable).

---

## Incident Response

- If a secret is exposed: rotate immediately, assess blast radius, document.
- If a vulnerability is found: classify severity, route to Numbuh 274, block deployment if CRITICAL/HIGH.
- If auth is bypassed: stop the mission, escalate, contain.

---

## Final Rule

Security is not a phase. It is not a review step. It is a property of every decision.

If a design creates attack surface, question it at design time.
If implementation handles secrets, verify at implementation time.
If deployment exposes services, check at deployment time.

Trust is useful. Verification is survival.

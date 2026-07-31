---
name: api-design
description: HTTP/REST API design — resource modeling, pagination, versioning, idempotency, consistent error shapes, and status code usage.
---

# API Design

## Resource Modeling

- URLs are nouns (resources), not verbs: `/users/123`, not `/getUser?id=123`.
- Use plural nouns: `/orders`, `/orders/{id}`.
- Nest for clear ownership: `/users/{id}/orders` (user's orders).
- Limit nesting to 2 levels — deeper means you need a top-level resource.
- Use kebab-case in URLs: `/order-items`, not `/orderItems`.

## HTTP Methods

| Method | Semantics | Idempotent | Safe |
|--------|-----------|-----------|------|
| GET | Read resource(s) | Yes | Yes |
| POST | Create resource | No | No |
| PUT | Replace resource entirely | Yes | No |
| PATCH | Partial update | No | No |
| DELETE | Remove resource | Yes | No |

## Pagination

Use cursor-based pagination for large or frequently-changing collections:

```json
{
  "data": [],
  "pagination": {
    "next_cursor": "eyJpZCI6MTAwfQ==",
    "has_more": true,
    "page_size": 25
  }
}
```

- Default page size: 25. Max: 100. Reject sizes above max.
- Offset-based (`?page=3&size=25`) is acceptable for small, stable datasets.
- Always include total count or `has_more` so clients know when to stop.

## Versioning

- URL prefix: `/v1/users` — simple, explicit, cache-friendly.
- Increment only on breaking changes (field removal, type change, semantic change).
- Non-breaking additions (new optional fields) do NOT require version bump.
- Support N-1 version for a deprecation period with clear sunset headers.

## Idempotency

- Clients send `Idempotency-Key` header on non-idempotent operations (POST).
- Server stores result keyed by (client, idempotency-key) for 24h.
- Duplicate requests return the stored result without re-executing.
- PUT and DELETE are naturally idempotent — same request, same outcome.

## Error Shape

Consistent error response across all endpoints:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Human-readable description",
    "details": [
      {"field": "email", "issue": "must be a valid email address"}
    ],
    "request_id": "req_abc123"
  }
}
```

- Always include `request_id` for traceability.
- `code` is machine-readable (stable, documented).
- `message` is human-readable (may change between versions).
- Use appropriate status codes: 400 client error, 401 unauthenticated, 403 unauthorized, 404 not found, 409 conflict, 422 unprocessable, 429 rate limited, 500 server error.

## Rate Limiting

- Return `429 Too Many Requests` with `Retry-After` header.
- Include rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.
- Rate limit by API key or authenticated user, not by IP alone.

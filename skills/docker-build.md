---
name: docker-build
description: Docker multi-stage builds — layer caching, minimal non-root images, .dockerignore, and CI integration patterns.
---

# Docker Build

## Multi-Stage Builds

Separate build dependencies from runtime to minimize image size:

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app ./cmd/server

# Stage 2: Runtime
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /app /usr/local/bin/app
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/app"]
```

Key principles:
- Build stage has compilers, tools, source — never ships to production.
- Runtime stage has only the binary and minimal OS dependencies.
- Final image is typically 10-50MB instead of 500MB+.

## Layer Caching

Order Dockerfile instructions from least-changing to most-changing:

1. Base image (rarely changes)
2. System packages (changes on upgrade)
3. Dependency manifest (`go.mod`, `package.json`) — changes when deps change
4. Dependency download (`go mod download`, `npm ci`) — cached until manifest changes
5. Source code copy — changes on every commit
6. Build command

## Non-Root Execution

Never run containers as root in production:

```dockerfile
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

Or use distroless (no shell, no user management needed):
```dockerfile
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app /app
USER nonroot:nonroot
```

## .dockerignore

Always include a `.dockerignore` to prevent bloating the build context:

```
.git
.gitignore
*.md
docs/
node_modules/
vendor/
bin/
dist/
.env*
*.test
coverage/
.github/
```

## Health Checks

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1
```

## CI Integration

- Tag images with git SHA for traceability: `myapp:abc1234`.
- Use `--cache-from` with registry images to speed up CI builds.
- Scan images for vulnerabilities: `trivy image myapp:latest`.
- Set `DOCKER_BUILDKIT=1` for better caching and parallel stage execution.

## Size Reduction Checklist

- [ ] Multi-stage build (do not ship build tools).
- [ ] Alpine or distroless base (not ubuntu/debian unless required).
- [ ] Strip binaries: `-ldflags="-s -w"` for Go.
- [ ] No dev dependencies in runtime stage.
- [ ] Combined RUN commands to reduce layers.
- [ ] `.dockerignore` excludes tests, docs, VCS files.

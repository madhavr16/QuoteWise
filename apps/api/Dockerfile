# syntax=docker/dockerfile:1

# Build stage. The minor-version tag tracks Go 1.24 patch releases on
# purpose (ADR-011); the Go version contract is enforced by go.mod.
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Runtime stage: small, non-root, exact pin.
FROM alpine:3.24.2
RUN adduser -D -u 10001 quotewise
COPY --from=build /out/api /usr/local/bin/api
USER quotewise
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/api"]
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
  CMD wget -q -O /dev/null http://localhost:8080/healthz || exit 1

# ============================================================================
# Stage 1: Build
# ============================================================================
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# C compiler for CGO (tree-sitter), git for module downloads
RUN apk add --no-cache git gcc musl-dev

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

ARG VERSION=dev

# CGO_ENABLED=1 required for tree-sitter
# No explicit GOOS/GOARCH — Docker buildx + QEMU handles target platform natively
RUN CGO_ENABLED=1 \
    go build \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/muti \
      ./cmd/muti

# ============================================================================
# Stage 2: Runtime — minimal image with only git (required for worktrees)
# ============================================================================
FROM alpine:3.20

RUN apk add --no-cache git ca-certificates

# Non-root user
RUN adduser -D -h /home/muti muti
USER muti
WORKDIR /workspace

COPY --from=builder /out/muti /usr/local/bin/muti

ENTRYPOINT ["muti"]
CMD ["--help"]

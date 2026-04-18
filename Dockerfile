# ============================================================================
# LazyDB — Multi-stage Docker build
# ============================================================================

# --- Stage 1: Builder -------------------------------------------------------
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/aymenworks/lazydb/cmd.version=${VERSION}" \
    -trimpath \
    -o /usr/local/bin/lazydb \
    .

# --- Stage 2: Runtime -------------------------------------------------------
FROM alpine:3.20

LABEL maintainer="Aymen <aymen@aymenworks.com>"
LABEL description="LazyDB — Universal Database TUI"
LABEL org.opencontainers.image.source="https://github.com/aymenworks/lazydb"
LABEL org.opencontainers.image.url="https://github.com/aymenworks/lazydb"

ARG VERSION=dev
LABEL org.opencontainers.image.version="${VERSION}"

# ca-certificates needed for TLS connections to remote databases
RUN apk add --no-cache ca-certificates

COPY --from=builder /usr/local/bin/lazydb /usr/local/bin/lazydb

ENTRYPOINT ["/usr/local/bin/lazydb"]

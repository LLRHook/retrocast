FROM golang:1.25-alpine AS go-toolchain

# Build Go and Rust in the same Alpine/libc environment as the final image.
FROM rust:1.99.1-alpine3.21 AS builder
COPY --from=go-toolchain /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}" CGO_ENABLED=1
RUN apk add --no-cache musl-dev make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make native \
    && go build -ldflags='-linkmode=external -extldflags=-static' -o bin/retrocast ./cmd/retrocast \
    && go build -ldflags='-linkmode=external -extldflags=-static' -o bin/permission-probe ./scripts/testdata/permissionprobe \
    && cd /tmp && /app/bin/permission-probe

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl \
    && adduser -D -u 1001 appuser
COPY --from=builder /app/bin/retrocast /usr/local/bin/retrocast
COPY --from=builder /app/migrations /migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1
CMD ["retrocast"]

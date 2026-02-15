FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o retrocast ./cmd/retrocast

FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl \
    && adduser -D -u 1001 appuser
COPY --from=builder /app/retrocast /usr/local/bin/retrocast
COPY --from=builder /app/migrations /migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1
CMD ["retrocast"]

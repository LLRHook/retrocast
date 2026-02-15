FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o retrocast ./cmd/retrocast

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/retrocast /usr/local/bin/retrocast
COPY --from=builder /app/migrations /migrations
EXPOSE 8080
CMD ["retrocast"]

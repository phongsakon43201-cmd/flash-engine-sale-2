# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod ./
RUN go mod download || true

# Copy source code
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api-bin ./cmd/api/main.go

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/worker-bin ./cmd/worker/main.go

# Production API Stage
FROM alpine:latest AS api
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/api-bin /app/api-bin
COPY --from=builder /app/.env /app/.env
EXPOSE 8080
CMD ["/app/api-bin"]

# Production Worker Stage
FROM alpine:latest AS worker
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/worker-bin /app/worker-bin
COPY --from=builder /app/.env /app/.env
CMD ["/app/worker-bin"]

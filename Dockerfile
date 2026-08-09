# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/api-bin ./cmd/api

# Build Worker binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/worker-bin ./cmd/worker

# Production API Stage
FROM alpine:3.22 AS api
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder --chown=app:app /app/api-bin /app/api-bin
COPY --from=builder --chown=app:app /app/web /app/web
USER app
EXPOSE 8080
CMD ["/app/api-bin"]

# Production Worker Stage
FROM alpine:3.22 AS worker
RUN apk add --no-cache ca-certificates tzdata && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=builder --chown=app:app /app/worker-bin /app/worker-bin
USER app
CMD ["/app/worker-bin"]

# ---- Build stage ----
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags "-s -w" -o /bin/api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Non-root user
RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

# Copy binary and migration files
COPY --from=builder /bin/api /app/api
COPY --from=builder /src/migrations /app/migrations

# Assets directory (email templates, etc.) — copy if present
COPY --from=builder /src/assets /app/assets

USER app

# Render injects PORT at runtime; default to 8080
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/app/api"]

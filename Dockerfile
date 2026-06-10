# ==============================================
# Sales Service - Multi-stage Dockerfile
# ==============================================

# ==============================================
# Stage 1: Dependencies and cache optimization
# ==============================================
FROM golang:1.24-alpine AS deps
WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Configure private Go modules
ARG GITHUB_TOKEN
ENV GOPRIVATE=github.com/mercadocercano/*
RUN if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ==============================================
# Stage 2: Build stage
# ==============================================
FROM deps AS builder

COPY . .

# Build optimized binary with security hardening
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -trimpath \
    -o sales-service .

# ==============================================
# Stage 3: Development stage (with hot reload)
# ==============================================
FROM mercado-cercano/go-dev:1.24 AS development

WORKDIR /app

# Configure private Go modules
ARG GITHUB_TOKEN
RUN if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Create necessary directories and set permissions
RUN mkdir -p tmp scripts uploads logs /go/pkg/mod && \
    chmod -R 777 /go/pkg && \
    chown -R appuser:appgroup /app tmp scripts uploads logs

# Copy source code with correct ownership
COPY --chown=appuser:appgroup . .

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

CMD sh -c 'if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi && air -c .air.toml'

# ==============================================
# Stage 4: Migrate stage (Alpine + psql para Job K8s)
# ==============================================
FROM alpine:3.18 AS migrate

RUN apk add --no-cache postgresql-client

WORKDIR /app
COPY --from=builder /app/migrations ./migrations

# ==============================================
# Stage 5: Production stage (Distroless)
# ==============================================
FROM gcr.io/distroless/static-debian12:nonroot AS production

# Metadata
LABEL org.opencontainers.image.title="Sales Service" \
      org.opencontainers.image.description="Sales Management Service for SaaS MT" \
      org.opencontainers.image.source="https://github.com/saas-mt/sales-service" \
      org.opencontainers.image.vendor="SaaS MT Team" \
      org.opencontainers.image.licenses="MIT"

WORKDIR /app

# Copy binary only
COPY --from=builder --chown=nonroot:nonroot /app/sales-service ./

# Use distroless nonroot user (uid=65532)
USER nonroot

EXPOSE 8080

ENTRYPOINT ["./sales-service"]

# ==============================================
# Default stage: Development
# ==============================================
FROM development

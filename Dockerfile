# ==============================================
# Order Service - Multi-stage Dockerfile
# ==============================================

# ==============================================
# Stage 1: Dependencies and cache optimization
# ==============================================
FROM golang:1.22-alpine AS deps
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy dependency files and download modules
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# ==============================================
# Stage 2: Build stage
# ==============================================
FROM deps AS builder

# Copy source code
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
FROM golang:1.22-alpine AS development

# Security: Create non-root user first
RUN addgroup -g 1001 -S appgroup && \
    adduser -S -D -h /app -s /bin/sh -G appgroup -u 1001 appuser

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    postgresql-client \
    git \
    && cp /usr/share/zoneinfo/UTC /etc/localtime \
    && echo "UTC" > /etc/timezone \
    && apk del tzdata

WORKDIR /app

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

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080

# Use go run directly (simple fix)
CMD ["go", "run", "."]

# ==============================================
# Stage 4: Production stage (Distroless)
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

# Multi-stage Dockerfile for OSINTMCP
# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

# Copy frontend package files
COPY web/package*.json ./
RUN npm ci

# Copy frontend source
COPY web/ ./

# Build frontend
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY . .

# Build backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 3: Runtime image
FROM debian:bookworm-slim

# Install runtime dependencies only (no Playwright/Chromium needed for RSS-only)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy backend binary from builder
COPY --from=backend-builder /app/server .

# Copy frontend build from builder
COPY --from=frontend-builder /app/web/dist ./web/dist

# Copy migrations
COPY migrations ./migrations

# Run as non-root user
RUN groupadd -g 1000 stratint && \
    useradd -m -u 1000 -g stratint stratint && \
    chown -R stratint:stratint /app

USER stratint

# Expose port (Cloud Run sets PORT env var)
ENV PORT=8080
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT}/healthz || exit 1

# Start server
CMD ["./server"]

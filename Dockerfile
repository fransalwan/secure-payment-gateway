# ==============================================================================
# Stage 1: Build Mobile-First Vue 3 + TypeScript + shadcn-vue (Reka UI) SPA
# ==============================================================================
FROM node:22-alpine AS frontend-builder

WORKDIR /build/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
# vite.config.ts outputs compiled bundle directly to ../web/dist (/build/web/dist)
RUN npm run build

# ==============================================================================
# Stage 2: Build Go API Server & Standalone Redis/Outbox Background Worker
# ==============================================================================
FROM golang:alpine AS backend-builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=auto

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

RUN go build -trimpath -ldflags="-s -w" -o /out/spg-api ./cmd/api \
    && go build -trimpath -ldflags="-s -w" -o /out/spg-worker ./cmd/worker

# ==============================================================================
# Stage 3: Minimal Hardened Production Runtime (Alpine + Non-Root User)
# ==============================================================================
FROM alpine:3.21 AS runtime

RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S -g 10001 spg \
    && adduser -S -u 10001 -G spg spg

ENV TZ=Asia/Jakarta \
    APP_ENV=production \
    APP_PORT=8080

WORKDIR /app

COPY --from=backend-builder /out/spg-api /app/spg-api
COPY --from=backend-builder /out/spg-worker /app/spg-worker
COPY --from=frontend-builder /build/web/dist /app/web/dist
COPY migrations/ /app/migrations/

RUN chown -R spg:spg /app

USER spg

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

CMD ["/app/spg-api"]


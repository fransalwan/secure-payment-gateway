# ==============================================================================
# Stage 1: Build Go API Server & Standalone Redis/Outbox Background Worker
# (Frontend SPA is pre-built in web/dist to avoid OOM on 1-2GB RAM Cloud VPS)
# ==============================================================================
FROM golang:alpine AS backend-builder

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=auto \
    GOGC=50

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

RUN go build -p 2 -trimpath -ldflags="-s -w" -o /out/spg-api ./cmd/api \
    && go build -p 2 -trimpath -ldflags="-s -w" -o /out/spg-worker ./cmd/worker

# ==============================================================================
# Stage 2: Minimal Hardened Production Runtime (Alpine + Non-Root User)
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
COPY web/dist /app/web/dist
COPY migrations/ /app/migrations/

RUN chown -R spg:spg /app

USER spg

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1

CMD ["/app/spg-api"]


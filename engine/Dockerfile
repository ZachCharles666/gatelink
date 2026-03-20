# ---- Stage 1: Build ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 先复制依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译（CGO 关闭，生成静态二进制）
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /bin/engine ./cmd/engine/

# ---- Stage 2: Runtime (alpine + ca-certs) ----
# 使用 alpine 而非 distroless，便于排查问题（生产可换 distroless）
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

# 不以 root 运行
RUN addgroup -S engine && adduser -S engine -G engine
USER engine

WORKDIR /app

COPY --from=builder /bin/engine /app/engine

# 迁移文件（随镜像打包，启动时可选执行）
COPY --from=builder /app/internal/db/migrations /app/migrations

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8081/health || exit 1

ENTRYPOINT ["/app/engine"]

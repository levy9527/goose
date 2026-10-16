# ============================================================
# Goose 演示 · Docker 化（Go 迁移）
#
# 关键：Go 编译出的是静态二进制，goose + Go 迁移 + 驱动全部打进二进制。
# 构建阶段才需要 Go；运行镜像里只需一个二进制文件 + 迁移 .sql 文件。
#
# 构建（-f 指定本文件，构建上下文仍是仓库根）：
#   docker build -f demo/Dockerfile -t goose-demo .
# 运行（以 SQLite 目标库的 up 命令为例）：
#   docker run --rm \
#     -e PG_DSN='postgres://postgres:postgres@host.docker.internal:5432/goose_demo?sslmode=disable' \
#     --add-host=host.docker.internal:host-gateway \
#     goose-demo up
# ============================================================

# ---- Stage 1: 构建静态二进制（只有这一步需要 Go）----
# 用镜像源(daocloud)拉取基础镜像，避免 docker hub 在国内连不上
FROM docker.m.daocloud.io/golang:1.25-alpine AS build
# 容器内也走 goproxy（goproxy.cn），否则默认 proxy.golang.org 可能拉不下来
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 纯 Go 驱动(modernc sqlite / pgx)，CGO_ENABLED=0 得到纯静态二进制
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -o /out/demo-sqlite ./demo/runner/sqlite

# ---- Stage 2: 运行时镜像（无 Go、无依赖安装）----
# 与 Stage 1 一致走 daocloud 镜像源，避免 docker hub 连不上
FROM docker.m.daocloud.io/alpine:3.20
# HTTPS 请求(拉 /posts)需要 CA 证书；这也是运行时唯一装的包
RUN apk add --no-cache ca-certificates

# 只要二进制 + 迁移 .sql 文件（Go 迁移已编译进二进制）
COPY --from=build /out/demo-sqlite /usr/local/bin/demo-sqlite
COPY demo/sqlite_migrations /app/demo/sqlite_migrations

WORKDIR /app
ENTRYPOINT ["demo-sqlite"]
# 默认执行 up，可覆盖：docker run ... goose-demo {up|status|show|reset}
CMD ["up"]

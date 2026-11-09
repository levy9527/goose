#!/bin/sh
#
# 编一个「只带 Postgres 驱动」的 goose CLI，供 Dockerfile.amd64 / Dockerfile.arm64 共用。
#
# 产物：bin/goose-linux-<arch>（bin/ 已在 .gitignore 中）
# 用法：
#   ./build-pg-goose.sh            # 默认 amd64
#   ./build-pg-goose.sh arm64      # 或 Go 认的任何 GOARCH
#   VERSION=v3.1.0 ./build-pg-goose.sh arm64
#
# 注意这里刻意用 POSIX sh（不是 bash）：alpine 基础镜像里没有 bash，
# Dockerfile 构建阶段会直接 `sh ./build-pg-goose.sh <arch>` 调它。
#
# 其它要点：
#   - CGO_ENABLED=0 + GOOS=linux → 纯静态二进制，能直接跑在 alpine 里
#   - 不设 GOOS/GOARCH 会编出宿主机平台（如 macOS arm64）的二进制，进容器跑不起来
#   - 驱动 tag 名单见 `make list-build-tags`；是 no_sqlite3 不是 no_sqlite
#     （Go 对未知 build tag 不报错，写错会静默失效并白白增大体积）
set -eu

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-${1:-amd64}}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

BUILD_TAGS="no_sqlite3,no_clickhouse,no_mssql,no_mysql,no_vertica,no_ydb,no_libsql,no_azuresql"

OUT="bin/goose-${GOOS}-${GOARCH}"

cd "$(dirname "$0")"
mkdir -p bin

echo "building $OUT (version=$VERSION tags=$BUILD_TAGS)"

CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION}" \
  -tags="$BUILD_TAGS" \
  -o "$OUT" ./cmd/goose

ls -lh "$OUT"

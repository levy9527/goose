# Goose 演示：自动化 SQL + 复杂 Go 代码（Postgres → SQLite 迁移）

这个 demo 展示 Goose 的核心能力：**把 Go 代码当成一等公民的 migration，
与 SQL migration 混在同一个「有序、可回滚」的迁移管线里执行。**

它复刻了一个真实的工程困境——**数据库迁移 + 联网拉数 + 改表结构**，三种步骤相互依赖，
只有顺序正确才能成功。

---

## 场景

```
Postgres.app_user（静态种子数据初始化）
   │  Go 迁移跨库搬数据
   ▼
SQLite.app_user（迁移结果）
   │
   ▼
SQLite.post 建表（SQL，含列 title）
   │  Go 迁移运行时拉取填充
   ▼
HTTP /users/1/posts ──▶ SQLite.post（title）
   │  SQL 迁移重命名列（最后一步）
   ▼
SQLite.post.heading
```

> PG 源库的 `app_user` 数据为**静态种子数据**，初始化时一次性写入（来源已确定），
> 演示运行时不再请求。仅 `post` 表的填充保留**运行时 HTTP**（Go 迁移）。

两个数据库各有一套 goose 迁移（文件名是 `goose create` 生成的**时间戳**，天然避免多人协作的序号冲突）：

**`demo/pg_migrations/` —— Postgres 源库（准备工作，全 SQL）**
| 文件 | 类型 | 作用 |
|---|---|---|
| `*_create_app_user.sql` | SQL | 建 `app_user` 表 |
| `*_seed_app_user.sql` | SQL | 写入 10 条静态种子数据 |

**`demo/sqlite_migrations/` —— SQLite 目标库（正式演示）**
| 文件 | 类型 | 作用 |
|---|---|---|
| `*_create_app_user.sql` | SQL | 建同构 `app_user` |
| `*_copy_app_user_from_pg.go` | **Go** | 连 PG → 读全部 user → 写进 SQLite（核心）|
| `*_create_post.sql` | SQL | 建 `post` 表 |
| `*_fetch_posts.go` | **Go**（NoTx 事务外）| 运行时 GET `/users/1/posts` → 填充 `post`（写旧列 `title`）|
| `*_rename_post_title.sql` | SQL | `ALTER TABLE post RENAME COLUMN title TO heading` |

> 关键点：`fetch_posts` 写入的是旧列名 `title`，而 `rename` 把它改成了 `heading`。
> **Goose 保证 `fetch_posts` 先于 `rename` 执行**——这就是它胜过「只跑脚本的工具」的地方。

---

## 运行

前提：本机 127.0.0.1:5432 有可用的 Postgres（用户/密码 `postgres/postgres`），
且能访问外网 jsonplaceholder（post 拉数需联网）。连接串可用环境变量覆盖：

```bash
export PG_DSN='postgres://postgres:postgres@127.0.0.1:5432/goose_demo?sslmode=disable'
```

### 好路径（Goose 自动化）

```bash
./demo/run_demo.sh
```

它依次：
1. `go run ./demo/runner/pg up` —— 重建 PG 库并跑迁移（SQL 建表 + SQL 静态种子）。
2. `go run ./demo/runner/sqlite up` —— 重建 SQLite 库并跑迁移：建表 → **Go 跨库搬数据** → 建 post → Go 运行时拉 `/posts` → 重命名列。
3. 打印两个库最终的数据，证明全部成功。

### 坏路径（非自动化：脚本/人肉先改表，Go 代码最后跑）

```bash
./demo/run_bad.sh
```

模拟「只自动化脚本、自动化不了代码」的工具：先把表建好、把列重命名好，最后才单独跑 Go 拉数程序
（它仍写旧列 `title`）→ **报错 `no such column: title`**。这正是 Goose 帮你避免的故障。

---

## 容器化部署（服务器只有 Docker、没有 Go）

Go 编译出来的是**静态二进制**，goose + 你的 Go 迁移 + 驱动全部打包在里面。
所以 Go 只需要在**构建**时存在（开发机或 CI）；服务器镜像里只有一个二进制，
**不用装 Go、不用 `go mod download`、不用装依赖**。

```bash
# 1) 编出静态 Linux 二进制（开发机上执行，CGO_ENABLED=0 确保纯静态）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o demo-sqlite-linux-arm64 ./demo/runner/sqlite

# 2) 打包进镜像（运行镜像只有二进制 + 迁移 .sql 文件）
docker build -t goose-demo .
docker run --rm \
  -e PG_DSN='postgres://postgres:postgres@host.docker.internal:5432/goose_demo?sslmode=disable' \
  --add-host=host.docker.internal:host-gateway \
  goose-demo sqlite up
```

### 两种镜像（均已本机构建验证，最终镜像无 Go）

**① 多阶段构建 `demo/Dockerfile`**（构建阶段用 golang 镜像现场编译）：
```bash
# 国内需用 daocloud 镜像源(基础镜像) + goproxy.cn(Go 依赖)，Dockerfile 已内置
docker build -f demo/Dockerfile -t goose-demo .

# 运行 SQLite 目标库迁移（连宿主 PG，拉 /posts 需外网）
docker run --rm -e PG_DSN='postgres://postgres:postgres@host.docker.internal:5432/goose_demo?sslmode=disable' \
  --add-host=host.docker.internal:host-gateway goose-demo up
```

**② 运行时验证镜像 `demo/Dockerfile.runtime`**（用预编译静态二进制，最快）：
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/demo-sqlite-linux-arm64 ./demo/runner/sqlite
docker build -f demo/Dockerfile.runtime -t goose-demo-runtime .
docker run --rm -e PG_DSN='postgres://postgres:postgres@host.docker.internal:5432/goose_demo?sslmode=disable' \
  --add-host=host.docker.internal:host-gateway goose-demo-runtime up
```

> 注意：每次 `docker run` 是临时容器，`up` 建的 `demo/demo.db` 不持久化。要看数据就
> `up`、`show` 放同一个容器（`--entrypoint sh -c "demo-sqlite up && demo-sqlite show"`），
> 或挂 volume 持久化。
>
> 国内网络：基础镜像用 `docker.m.daocloud.io` 源（golang/alpine），Go 依赖走 `GOPROXY=goproxy.cn`。

---

## 为什么用 `runner`（goose 当库），而不是 goose CLI？

goose CLI 只能执行**已编译进二进制的** Go migration；`-dir` 里未注册的 `.go` 文件会直接报错。
所以官方跑 Go migration 的标准姿势是「goose 当库用」：写一个很小的 main，**import 迁移包**（触发
`init()` 注册），再调用 `goose.UpContext(db, dir)`。

注意 goose 的 Go 迁移注册在**进程全局** map：若某个目录里没有 `.go` 文件，goose 会回退把
**全部已注册 Go 迁移**都塞进去。因此 PG（纯 SQL）与 SQLite（含 Go）拆成两个独立入口，
PG 入口不 import SQLite 迁移包：

```bash
go run ./demo/runner/pg up          # PG 源库（纯 SQL）
go run ./demo/runner/sqlite up      # SQLite 目标库（SQL + Go）
go run ./demo/runner/{pg|sqlite} {up|status|show|reset}
```

---

## 目录结构

```
demo/
  api/api.go                  # 共享“复杂 Go 代码”：HTTP 拉取 + 解码 + 插入
  pg_migrations/              # Postgres 源库迁移（全 SQL，含静态种子）
  sqlite_migrations/          # SQLite 目标库迁移（SQL + Go）
  runnerlib/runnerlib.go      # goose 当库用的共享执行逻辑
  runner/pg/main.go           # PG 源库入口（不 import SQLite 迁移）
  runner/sqlite/main.go       # SQLite 目标库入口（import SQLite 迁移）
  bad/main.go                 # 非自动化坏路径复现程序
  run_demo.sh                 # 好路径一键运行
  run_bad.sh                  # 坏路径一键运行
  demo.html                   # HTML 版演示说明（浏览器直接打开）
```

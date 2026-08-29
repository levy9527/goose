# Goose 演示：自动化 SQL + 复杂 Go 代码（Postgres → SQLite 迁移）

这个 demo 展示 Goose 的核心能力：**把 Go 代码当成一等公民的 migration，
与 SQL migration 混在同一个「有序、可回滚」的迁移管线里执行。**

它复刻了一个真实的工程困境——**数据库迁移 + 联网拉数 + 改表结构**，三种步骤相互依赖，
只有顺序正确才能成功。

---

## 场景

```
jsonplaceholder /users ──────────▶ PG.app_user        （准备工作，Go 迁移拉数据）
PG.app_user ────────────────────▶ SQLite.app_user     （跨库迁移，Go 迁移搬数据）
jsonplaceholder /users/1/posts ─▶ SQLite.post         （Go 迁移拉数据）
SQLite.post.title ─────────────▶ SQLite.post.heading  （SQL 迁移改表：重命名列）
```

两个数据库各有一套 goose 迁移（文件名是 `goose create` 生成的**时间戳**，天然避免多人协作的序号冲突）：

**`demo/pg_migrations/` —— Postgres 源库（准备工作）**
| 文件 | 类型 | 作用 |
|---|---|---|
| `*_create_app_user.sql` | SQL | 建 `app_user` 表 |
| `*_seed_users_from_api.go` | **Go** | HTTP GET `/users` → 反序列化 → 填充 PG |

**`demo/sqlite_migrations/` —— SQLite 目标库（正式演示）**
| 文件 | 类型 | 作用 |
|---|---|---|
| `*_create_app_user.sql` | SQL | 建同构 `app_user` |
| `*_copy_app_user_from_pg.go` | **Go** | 连 PG → 读全部 user → 写进 SQLite（核心）|
| `*_create_post.sql` | SQL | 建 `post` 表 |
| `*_fetch_posts.go` | **Go** | HTTP GET `/users/1/posts` → 填充 `post`（写旧列 `title`）|
| `*_rename_post_title.sql` | SQL | `ALTER TABLE post RENAME COLUMN title TO heading` |

> 关键点：`fetch_posts` 写入的是旧列名 `title`，而 `rename` 把它改成了 `heading`。
> **Goose 保证 `fetch_posts` 先于 `rename` 执行**——这就是它胜过「只跑脚本的工具」的地方。

---

## 运行

前提：本机 127.0.0.1:5432 有可用的 Postgres（用户/密码 `postgres/postgres`），且能访问外网
（jsonplaceholder）。连接串可用环境变量覆盖：

```bash
export PG_DSN='postgres://postgres:postgres@127.0.0.1:5432/goose_demo?sslmode=disable'
```

### 好路径（Goose 自动化）

```bash
./demo/run_demo.sh
```

它依次：
1. `go run ./demo/runner pg up` —— 重建 PG 库并跑迁移（SQL 建表 + Go 拉 `/users` 填充）。
2. `go run ./demo/runner sqlite up` —— 重建 SQLite 库并跑迁移：建表 → **Go 跨库搬数据** → 建 post → Go 拉 `/posts` → 重命名列。
3. 打印两个库最终的数据，证明全部成功。

### 坏路径（非自动化：脚本/人肉先改表，Go 代码最后跑）

```bash
./demo/run_bad.sh
```

模拟「只自动化脚本、自动化不了代码」的工具：先把表建好、把列重命名好，最后才单独跑 Go 拉数程序
（它仍写旧列 `title`）→ **报错 `no such column: title`**。这正是 Goose 帮你避免的故障。

---

## 为什么不用 goose CLI，而用 `runner`？

goose CLI 只能执行**已编译进二进制的** Go migration；`-dir` 里未注册的 `.go` 文件会直接报错。
所以官方跑 Go migration 的标准姿势是「goose 当库用」：写一个很小的 main，**import 迁移包**（触发
`init()` 注册），再调用 `goose.UpContext(db, dir)`。

`demo/runner/main.go` 就是这个 runner，支持：

```bash
go run ./demo/runner {pg|sqlite} {up|status|show|reset}
```

---

## 目录结构

```
demo/
  api/api.go                  # 共享“复杂 Go 代码”：HTTP 拉取 + 解码 + 插入
  pg_migrations/              # Postgres 源库迁移（准备）
  sqlite_migrations/          # SQLite 目标库迁移（正式演示）
  runner/main.go              # goose 当库用的迁移执行器
  bad/main.go                 # 非自动化坏路径复现程序
  run_demo.sh                 # 好路径一键运行
  run_bad.sh                  # 坏路径一键运行
```

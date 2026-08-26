# 本地开发指南

## 推荐开发环境

本地开发默认使用以下组合：

- Air 监听 Go 文件并自动重新构建、启动后端。
- SQLite 保存本地开发数据，不需要启动 Docker、PostgreSQL 或 Redis。
- Rsbuild 在独立终端中运行前端开发服务器，并把 `/api`、`/mj`、`/pg` 请求代理到后端。

需要 Go 1.25.1 或更高的兼容版本，以及 Bun。Air 已作为 Go 工具依赖固定在 `go.mod` 中，不需要全局安装。

## 启动后端

在仓库根目录运行：

```bash
go tool air
```

未设置 `SQL_DSN` 时，后端自动使用 SQLite，默认数据库文件为 `one-api.db`。如需指定其他位置，可以在启动 Air 前设置 `SQLITE_PATH`。

后端默认监听：

```text
http://localhost:3000
```

修改 Go 文件后，Air 会根据 `.air.toml` 自动重新构建并重启后端。首次构建时会自动创建被 Git 忽略的最小 `web/dist/index.html`，满足 Go 嵌入静态目录的构建要求；实际前端仍由 Rsbuild 提供。测试文件、前端目录、文档目录和本地缓存目录不会触发重启。

## 启动前端

打开另一个终端，在仓库根目录运行：

```bash
cd web
bun install
bun run dev -- --host 0.0.0.0 --port 5173
```

浏览器访问：

```text
http://localhost:5173
```

前端开发服务器默认把 API 请求代理到 `http://localhost:3000`。后续依赖没有变化时，可以跳过 `bun install`。

## Makefile 命令

安装 GNU Make 的环境也可以使用：

```bash
# Air + SQLite 后端
make dev-api

# 前后端并行启动
make dev
```

Docker/PostgreSQL 开发环境不再是默认方案。只有在验证 PostgreSQL、Redis 或容器行为时才使用：

```bash
make dev-api-docker
```

修改 Go 后端镜像后可以重新构建 Docker 服务：

```bash
make dev-api-rebuild
```

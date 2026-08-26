# 6789-AIGateway
6789自研的AI网关

## 本地开发

推荐使用 Air 热重载 Go 后端，并使用默认的本地 SQLite 数据库。后端和前端分别在两个终端中运行：

```bash
# 终端 1：仓库根目录
go tool air
```

```bash
# 终端 2：仓库根目录
cd web
bun install
bun run dev -- --host 0.0.0.0 --port 5173
```

前端访问地址为 `http://localhost:5173`，后端地址为 `http://localhost:3000`。完整说明见[本地开发指南](docs/development.md)。

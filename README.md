# watchend

单用户自托管的 GitHub Stars 管理应用。

## 功能

- 单用户用户名/密码登陆支持
- 同步管理用户的 Star 仓库，可按多种方式排序，支持升序/降序切换和分页加载；应用启动后会按配置间隔自动同步
- 支持前往随机星标仓库

## 配置

应用通过环境变量配置：

| 环境变量 | 必需 | 默认值 | 说明 |
|---|---:|---|---|
| `WATCHEND_DATABASE_PATH` | 否 | `/data/watchend.db` | SQLite 数据库路径 |
| `WATCHEND_ADMIN_USERNAME` | 否 | `admin` | 首次启动时创建管理员 |
| `WATCHEND_ADMIN_PASSWORD` | 是 | - | 首次启动时创建管理员密码；之后修改不会重置已有密码 |
| `WATCHEND_ADDR` | 否 | `:3000` | HTTP 监听地址 |
| `WATCHEND_SESSION_TTL` | 否 | `720h` | Session 有效期 |
| `WATCHEND_SECURE_COOKIES` | 否 | `true` | HTTPS 环境保持 `true`；本地 HTTP 开发设置为 `false` |
| `WATCHEND_GITHUB_TOKEN` | 是 | - | GitHub Personal Access Token |
| `WATCHEND_SYNC_INTERVAL` | 否 | `6h` | 后台自动同步间隔；应用启动后首次等待该间隔再同步 |

`WATCHEND_GITHUB_TOKEN` 用于调用 GitHub API，Token 应只授予同步所需的最小权限。

首次启动时，应用会在数据库所在目录创建 `session-secret` 文件，并设置为仅所有者可读写（`0600`）。该文件会在重启时复用；删除它会使现有登录 Session 失效。

## Docker

构建镜像：

```sh
docker build -t watchend .
```

启动示例：

```sh
docker run -d \
  --name watchend \
  -p 3000:3000 \
  -v ./watchend/data:/data \
  -e WATCHEND_SECURE_COOKIES='false' \
  -e WATCHEND_ADMIN_PASSWORD='replace-with-a-strong-password' \
  -e WATCHEND_GITHUB_TOKEN='your-github-token' \
  watchend
```

打开 `http://localhost:3000`，默认管理员用户名为 `admin`。

生产环境应部署在 HTTPS reverse proxy 后，并保持：

```sh
WATCHEND_SECURE_COOKIES=true
```

# MyUrls

基于 Go 1.24 与 Redis/SQLite 实现的本地短链接服务，用于缩短 URL 与短链接还原。

## 目录

- [特性介绍](#特性介绍)
- [版本迁移（v1 → v2）](#版本迁移v1--v2)
- [快速开始](#快速开始)
- [安装与构建](#安装与构建)
- [配置说明](#配置说明)
- [使用指南](#使用指南)
- [高级功能](#高级功能)
- [开发指南](#开发指南)
- [项目信息](#项目信息)

## 特性介绍

### 架构设计

项目采用标准的Go分层架构，便于维护和扩展：

```shell
MyUrls/
├── cmd/                     # 程序入口
│   ├── myurls/              # 主服务
│   └── sync_data/           # 数据同步工具
├── internal/                # 内部包
│   ├── config/              # 配置管理
│   ├── handler/             # HTTP处理器
│   ├── logger/              # 日志处理
│   ├── model/               # 数据模型
│   ├── service/             # 业务逻辑层
│   ├── storage/             # 存储层
│   └── utils/               # 工具函数
├── web/                     # 静态文件
├── data/                    # 数据文件目录
├── output/                   # 构建输出目录
└── logs/                    # 日志文件目录
```

**分层说明：**

- **表现层（handler）**：处理HTTP请求和响应
- **业务层（service）**：处理核心业务逻辑
- **存储层（storage）**：抽象存储接口，支持Redis和SQLite，内置LRU缓存
- **配置层（config）**：统一配置管理
- **工具层（utils）**：公用工具函数

### 存储与缓存

**存储支持：**

- **Redis**（推荐）：高性能缓存数据库，支持过期时间
- **SQLite**：嵌入式数据库，无需额外安装服务

**LRU缓存特性：**

- **线程安全**：支持并发访问，使用读写锁保证性能
- **TTL支持**：支持缓存项的过期时间设置
- **自动清理**：后台定期清理过期缓存项
- **容量控制**：超出容量时自动清理最少使用的缓存项
- **高性能**：基于双向链表和哈希表实现，O(1)时间复杂度

## 版本迁移（v1 → v2）

v2（tag `v2.0.0`）相对 v1（tag `v1.0.0`，对应 master 旧版）是一次架构级重构。从 v1 升级前请重点关注以下变更。

> **数据兼容性**：v1 与 v2 的 Redis 数据格式完全一致（同样使用 DB 0，键为短链接、值为原始 URL）。继续使用 Redis 时无需迁移任何数据，直接升级即可。

### 1. 配置方式（破坏性变更）

v1 的服务与存储参数通过命令行参数（`-domain`、`-conn`、`-password` 等）配置；迁移到 v2 时推荐统一改用**环境变量**配置，变量名也按模块重新分层命名。升级后的等价配置如下：

```bash
# 服务
export MYURLS_SERVER_DOMAIN=example.com            # 原 -domain / MYURLS_DOMAIN
export MYURLS_SERVER_PORT=8080                      # 原 -port   / MYURLS_PORT
export MYURLS_SERVER_PROTO=https                    # 原 -proto  / MYURLS_PROTO

# 存储（Redis）
export MYURLS_STORAGE_TYPE=redis
export MYURLS_STORAGE_REDIS_ADDR=localhost:6379     # 原 -conn     / MYURLS_REDIS_CONN
export MYURLS_STORAGE_REDIS_PASSWORD=your_password  # 原 -password / MYURLS_REDIS_PASSWORD

./myurls
```

新旧对照：

| 用途 | v1 | v2（环境变量） |
|------|----|----|
| 监听端口 | `-port` / `MYURLS_PORT` | `MYURLS_SERVER_PORT` |
| 服务域名 | `-domain` / `MYURLS_DOMAIN` | `MYURLS_SERVER_DOMAIN` |
| 协议 | `-proto` / `MYURLS_PROTO` | `MYURLS_SERVER_PROTO` |
| Redis 地址 | `-conn` / `MYURLS_REDIS_CONN` | `MYURLS_STORAGE_REDIS_ADDR` |
| Redis 密码 | `-password` / `MYURLS_REDIS_PASSWORD` | `MYURLS_STORAGE_REDIS_PASSWORD` |

> **可选：配置文件**　以上配置也可写入 `conf/app.toml`（详见[配置方式与优先级](#配置方式与优先级)）。优先级为 `环境变量 > conf/app.toml > 内置默认值`，环境变量会覆盖文件中的同名项，二者任选其一或混用均可。

### 2. 存储与缓存

- v1 仅支持 **Redis**；v2 新增 **SQLite** 存储，可通过 `MYURLS_STORAGE_TYPE=redis|sqlite` 切换。
- v2 内置可选的 **LRU 本地缓存层**（`MYURLS_STORAGE_CACHE_*` 配置），降低后端存储压力。
- 如需将 v1 的 Redis 数据迁移到 SQLite，使用新增的[数据同步工具](#数据同步工具)。

### 3. 新增能力

- **数据同步工具** `sync_data`：在 Redis 与 SQLite 之间批量同步数据（保留 TTL）。
- **缓存管理接口**：`DELETE /cache`、`GET /cache/clear`（需携带启动时生成的 token）。
- **安全加固**：修复短链接写入竞态（改用原子 `SET NX EX`）、补充 URL 校验等。

### 4. 目录结构调整

| 内容 | v1 | v2 |
|------|----|----|
| 代码组织 | 扁平结构（根目录下 `main.go`、`logic.go`、`handlers.go` 等） | 分层架构（`cmd/`、`internal/{config,handler,service,storage,...}`） |
| 静态文件 | `public/` | `web/` |
| 构建输出 | `build/` | `output/` |
| 配置文件 | 无 | `conf/app.toml` |

### 5. Docker 部署变更

- v1 通过 `entrypoint` 传入 `-conn myurls-redis:6379`；v2 改用环境变量 `MYURLS_STORAGE_REDIS_ADDR: myurls-redis:6379`。
- v2 需额外挂载配置目录 `./conf:/app/conf`，并使用新版环境变量名（如 `MYURLS_SERVER_PORT`）。
- 升级时请同步更新 `.env`，可参考最新的 [.env.example](.env.example)。

## 快速开始

### Docker方式（推荐）

使用Docker可以快速部署，无需安装其他依赖：

```shell
# 直接运行（通过环境变量配置）
docker run -d --restart always --name myurls \
  -v ./data:/app/data \
  -p 8080:8080 \
  -e MYURLS_SERVER_DOMAIN=example.com \
  careywong/myurls:latest

# 使用docker-compose
git clone https://github.com/CareyWang/MyUrls.git
cd MyUrls
cp .env.example .env
docker-compose up -d
```

### 二进制文件

前往 [Actions](https://github.com/CareyWang/MyUrls/actions/workflows/go.yml) 下载对应平台的可执行文件：

```bash
# 基本运行（通过环境变量配置）
export MYURLS_SERVER_DOMAIN=example.com
export MYURLS_SERVER_PORT=8080
./myurls

# 使用PM2守护进程
MYURLS_SERVER_DOMAIN=example.com pm2 start myurls --name myurls
```

## 安装与构建

### 环境要求

- Go 1.24+
- Redis（可选，推荐）
- SQLite（可选）

### 构建项目

```bash
# 克隆项目
git clone https://github.com/CareyWang/MyUrls.git
cd MyUrls

# 安装依赖
make install

# 构建主程序
make default

# 构建数据同步工具
make sync_data

# 构建所有平台
make all
```

**构建目标：**

- `make linux` - Linux AMD64
- `make darwin` - macOS AMD64  
- `make windows` - Windows x64
- `make aarch64` - Linux ARM64

### Redis安装（可选）

如果选择使用Redis存储：

```bash
# Ubuntu/Debian
sudo apt-get update
sudo add-apt-repository ppa:chris-lea/redis-server -y
sudo apt-get update
sudo apt-get install redis-server -y
```

## 配置说明

### 配置方式与优先级

v2 主服务**不接受命令行参数**，统一通过「配置文件 + 环境变量」配置（仅数据同步工具 `sync_data` 使用命令行参数）。三者优先级由低到高为：

```text
内置默认值  <  conf/app.toml  <  环境变量
```

因此配置文件与环境变量**任选其一即可**：配置文件可选（缺失时回退到默认值），环境变量会逐项覆盖配置文件中的同名项，也可二者混用（用环境变量临时覆盖文件中的个别配置）。

完整示例见 [`conf/app.toml`](conf/app.toml)：

```toml
[server]
port = "8080"
domain = "localhost:8080"
proto = "https"

[storage]
type = "redis"               # redis | sqlite
redis_addr = "localhost:6379"
redis_password = ""
sqlite_file = "./data/myurls.db"
cache_enabled = false
cache_size = 128
cache_ttl = 300              # 单位：秒
```

### 环境变量

```bash
# 服务配置
export MYURLS_SERVER_PORT=8080
export MYURLS_SERVER_DOMAIN=localhost:8080
export MYURLS_SERVER_PROTO=https

# 存储配置
export MYURLS_STORAGE_TYPE=redis # redis|sqlite
export MYURLS_STORAGE_REDIS_ADDR=localhost:6379
export MYURLS_STORAGE_REDIS_PASSWORD=password
export MYURLS_STORAGE_SQLITE_FILE=./data/myurls.db

# 缓存配置
export MYURLS_STORAGE_CACHE_ENABLED=true      # 是否启用缓存，默认true
export MYURLS_STORAGE_CACHE_SIZE=128          # 缓存容量，默认128
export MYURLS_STORAGE_CACHE_TTL=300           # 缓存过期时间，默认5m
```

## 使用指南

### 基本使用

```bash
# 使用SQLite（无需额外依赖）
export MYURLS_SERVER_DOMAIN=example.com
export MYURLS_STORAGE_TYPE=sqlite
./myurls

# 使用Redis
export MYURLS_SERVER_DOMAIN=example.com
export MYURLS_STORAGE_TYPE=redis
export MYURLS_STORAGE_REDIS_ADDR=localhost:6379
./myurls
```

### API接口

访问 `http://your-domain:port` 即可使用Web界面创建和管理短链接。

## 高级功能

### 数据同步工具

项目提供数据同步工具，用于在Redis和SQLite之间同步数据：

```bash
# 基本用法
./output/sync_data -redis-addr localhost:6379 -sqlite-file ./data/myurls.db

# 使用环境变量
export SYNC_REDIS_ADDR=localhost:6379
export SYNC_REDIS_PASSWORD=your_password
export SYNC_SQLITE_FILE=./data/myurls.db
./output/sync_data

# 自定义批量大小
./output/sync_data -batch-size 500
```

**功能特性：**

- 从Redis批量读取所有短链接数据
- 自动保持TTL（过期时间）信息
- 批量写入SQLite数据库
- 支持进度显示和错误重试

### 日志管理

配置日志自动轮转和清理（假定工作目录为 `/app`）：

```bash
# 创建logrotate配置
tee > /etc/logrotate.d/myurls <<EOF
/app/logs/access.log {
    daily
    rotate 7
    missingok
    notifempty
    compress
    delaycompress
    copytruncate
    create 640 root adm
}
EOF

# 测试配置
logrotate -d /etc/logrotate.d/myurls
```

## 开发指南

### 开发环境

```bash
# 安装依赖
go mod tidy

# 运行测试
go test ./...

# 跳过Redis相关测试（如果没有Redis服务）
SKIP_REDIS_TESTS=1 go test ./...

# 格式化代码
make fmt

# 清理构建文件
make clean
```

### 贡献指南

PRs accepted. 如果编辑README，请符合 [standard-readme](https://github.com/RichardLitt/standard-readme) 规范。

## 项目信息

### 维护者

[@CareyWang](https://github.com/CareyWang)

### 许可证

MIT © 2025 CareyWang

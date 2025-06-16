# MyUrls

基于 Go 1.24 与 Redis/SQLite 实现的本地短链接服务，用于缩短 URL 与短链接还原。

## 目录

- [特性介绍](#特性介绍)
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
├── build/                   # 构建输出目录
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

## 快速开始

### Docker方式（推荐）

使用Docker可以快速部署，无需安装其他依赖：

```shell
# 直接运行
docker run -d --restart always --name myurls \
  -v ./data:/app/data \
  -p 8080:8080 \
  careywong/myurls:latest \
  -domain example.com

# 使用docker-compose
git clone https://github.com/CareyWang/MyUrls.git
cd MyUrls
cp .env.example .env
docker-compose up -d
```

### 二进制文件

前往 [Actions](https://github.com/CareyWang/MyUrls/actions/workflows/go.yml) 下载对应平台的可执行文件：

```bash
# 基本运行
./myurls -domain example.com -port 8080

# 使用PM2守护进程
pm2 start myurls --name myurls -- -domain example.com
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

### 命令行参数

```bash
Usage of ./myurls:
  -domain string    服务域名 (default "localhost:8080")
  -port string      监听端口 (default "8080")
  -proto string     协议类型 (default "https")
  -h               显示帮助
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
export MYURLS_STORAGE_TYPE=sqlite
./myurls -domain example.com -port 8080

# 使用Redis
export MYURLS_STORAGE_TYPE=redis
export MYURLS_STORAGE_REDIS_ADDR=localhost:6379
./myurls -domain example.com -port 8080
```

### API接口

访问 `http://your-domain:port` 即可使用Web界面创建和管理短链接。

## 高级功能

### 数据同步工具

项目提供数据同步工具，用于在Redis和SQLite之间同步数据：

```bash
# 基本用法
./build/sync_data -redis-addr localhost:6379 -sqlite-file ./data/myurls.db

# 使用环境变量
export SYNC_REDIS_ADDR=localhost:6379
export SYNC_REDIS_PASSWORD=your_password
export SYNC_SQLITE_FILE=./data/myurls.db
./build/sync_data

# 自定义批量大小
./build/sync_data -batch-size 500
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

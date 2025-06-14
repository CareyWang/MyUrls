# MyUrls

基于 Go 1.24 与 Redis/SQLite 实现的本地短链接服务，用于缩短 URL 与短链接还原。

## Table of Contents

- [项目结构](#项目结构)
- [Dependencies](#dependencies)
  - [Docker](#docker)
  - [Install](#install)
  - [Usage](#usage)
    - [日志清理](#日志清理)
  - [Maintainers](#maintainers)
  - [Contributing](#contributing)
  - [License](#license)

## 项目结构

项目采用标准的Go项目分层架构，便于维护和扩展：

```
MyUrls/
├── cmd/myurls/                # 主程序入口
│   └── main.go
├── internal/                  # 内部包，不对外暴露
│   ├── config/               # 配置管理
│   │   ├── config.go
│   │   └── config_test.go
│   ├── handler/              # HTTP处理器
│   │   └── url.go
│   ├── logger/               # 日志处理
│   │   └── logger.go
│   ├── model/                # 数据模型和常量
│   │   └── response.go
│   ├── service/              # 业务逻辑层
│   │   ├── url.go
│   │   └── url_test.go
│   ├── storage/              # 存储层
│   │   ├── interface.go      # 存储接口定义
│   │   ├── manager.go        # 存储管理器
│   │   ├── redis.go          # Redis驱动
│   │   └── sqlite.go         # SQLite驱动
│   └── utils/                # 工具函数
│       └── random.go
├── web/                      # 静态文件
│   ├── index.html
│   ├── favicon.ico
│   └── logo.png
├── scripts/                  # 构建脚本
├── data/                     # 数据文件目录
├── build/                    # 构建输出目录
└── logs/                     # 日志文件目录
```

**架构分层说明：**
- **表现层（handler）**：处理HTTP请求和响应
- **业务层（service）**：处理核心业务逻辑
- **存储层（storage）**：抽象存储接口，支持Redis和SQLite
- **配置层（config）**：统一配置管理
- **工具层（utils）**：公用工具函数

# Dependencies

本服务支持两种存储方式：

1. **Redis**（推荐）：高性能缓存数据库，支持过期时间
2. **SQLite**：嵌入式数据库，无需额外安装服务

## 存储配置

通过环境变量配置存储类型：

```bash
# 使用Redis（默认）
export MYURLS_STORAGE_TYPE=redis
export MYURLS_REDIS_CONN=localhost:6379
export MYURLS_REDIS_PASSWORD=yourpassword

# 使用SQLite
export MYURLS_STORAGE_TYPE=sqlite
export MYURLS_SQLITE_FILE=./data/myurls.db
```

### Redis安装（可选）

如果选择使用Redis存储，需要安装Redis服务：

```shell script
sudo apt-get update

# 安装Redis
sudo add-apt-repository ppa:chris-lea/redis-server -y 
sudo apt-get update 
sudo apt-get install redis-server -y 
```

## Docker 

现在你可以无需安装其他服务，使用 docker 或 [docker-compose](https://docs.docker.com/compose/install/) 部署本项目。注：请自行修改 .env 中参数。

```
docker run -d --restart always --name myurls careywong/myurls:latest -domain example.com -port 8002 -conn 127.0.0.1:6379 -password ''
```

```shell script
git clone https://github.com/CareyWang/MyUrls.git MyUrls

cd MyUrls
cp .env.example .env

docker-compose up -d
```

## Install

安装项目依赖

```shell script
make install
```

生成可执行文件，目录位于 build/ 。默认当前平台，其他平台请参照 Makefile 或执行对应 go build 命令。

```shell script
make
```

## Usage

前往 [Actions](https://github.com/CareyWang/MyUrls/actions/workflows/go.yml) 下载对应平台可执行文件。

### 运行参数

```shell script
Usage of ./myurls:
  -domain string
        domain of the server (default "localhost:8080")
  -h    display help
  -port string
        port to run the server on (default "8080")
  -proto string
        protocol of the server (default "https")
```

### 环境变量配置

除了命令行参数，还支持通过环境变量配置：

```bash
# 服务配置
export MYURLS_PORT=8080
export MYURLS_DOMAIN=localhost:8080
export MYURLS_PROTO=https

# 存储配置
export MYURLS_STORAGE_TYPE=redis|sqlite
export MYURLS_REDIS_CONN=localhost:6379
export MYURLS_REDIS_PASSWORD=password
export MYURLS_SQLITE_FILE=./data/myurls.db
```

### 运行示例

```bash
# 使用SQLite（无需额外依赖）
./myurls -domain example.com -port 8080

# 使用Redis
export MYURLS_STORAGE_TYPE=redis
export MYURLS_REDIS_CONN=localhost:6379
./myurls -domain example.com -port 8080
```

建议配合 [pm2](https://pm2.keymetrics.io/) 开启守护进程。

```shell script
pm2 start myurls --name myurls -- -domain example.com
```

### 日志清理

假定工作目录为 `/app`，可基于 logrotate 配置应用日志的自动轮转与清理。可参考示例配置，每天轮转一次日志文件，保留最近7天

```shell 
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

# 测试是否正常工作，不会实际执行切割
logrotate -d /etc/logrotate.d/myurls
```

## Development

### 开发环境

```bash
# 克隆项目
git clone https://github.com/CareyWang/MyUrls.git
cd MyUrls

# 安装依赖
go mod tidy

# 构建项目
make build

# 运行测试
go test ./...

# 跳过Redis相关测试（如果没有Redis服务）
SKIP_REDIS_TESTS=1 go test ./...
```

### 项目构建

```bash
# 构建当前平台可执行文件
make default

# 构建所有平台
make all

# 构建特定平台
make linux    # Linux AMD64
make darwin   # macOS AMD64  
make windows  # Windows x64
make aarch64  # Linux ARM64
```

### 代码格式化

```bash
make fmt
```

## Maintainers

[@CareyWang](https://github.com/CareyWang)

## Contributing

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © 2024 CareyWang

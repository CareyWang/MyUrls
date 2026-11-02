# MyUrls

MyUrls 是基于 Go 与 Redis 构建的短链接服务，提供长链接缩短、用户指定短键以及短链接重定向功能。前端页面与静态资源通过 Go `embed` 特性编译至单个二进制文件中，无需单独分发静态资源文件。

## 目录

- [前置要求](#前置要求)
- [快速开始](#快速开始)
  - [使用 Docker Compose 部署](#使用-docker-compose-部署)
  - [使用 Docker 运行](#使用-docker-运行)
  - [使用二进制文件运行](#使用二进制文件运行)
- [配置说明](#配置说明)
- [接口说明](#接口说明)
  - [创建短链接](#创建短链接)
  - [访问短链接](#访问短链接)
  - [Web 界面](#web-界面)
- [核心机制](#核心机制)
  - [短键生成与冲突重试](#短键生成与冲突重试)
  - [有效期与防覆盖规则](#有效期与防覆盖规则)
- [构建与编译](#构建与编译)
- [日志与维护](#日志与维护)
- [测试](#测试)
- [许可证](#许可证)

## 前置要求

运行 MyUrls 前，需要准备以下环境：

- **Redis 服务**：用于存储长短链接的映射关系。
- 若使用容器化部署，需要安装 Docker 及 Docker Compose。
- 若从源码编译，需要安装 Go 1.27 或更高版本。

## 快速开始

### 使用 Docker Compose 部署

推荐使用 Docker Compose 快速启动包含 Redis 依赖的完整服务。

1. 克隆代码仓库：

   ```bash
   git clone https://github.com/CareyWang/MyUrls.git
   cd MyUrls
   ```

2. 复制并编辑环境变量配置文件：

   ```bash
   cp .env.example .env
   ```

   根据实际需求修改 `.env` 中的参数：

   ```env
   MYURLS_PORT=8080
   MYURLS_DOMAIN=example.com
   MYURLS_PROTO=https
   ```

3. 启动服务：

   ```bash
   docker-compose up -d
   ```

服务启动后，监听端口为 `.env` 中配置的 `MYURLS_PORT`（默认 `8080`），应用日志持久化保存至宿主机 `./data/myurls/logs` 目录。

### 使用 Docker 运行

如果已有现成的 Redis 实例，可以直接运行 Docker 镜像：

```bash
docker run -d \
  --name myurls \
  --restart always \
  -p 8080:8080 \
  careywong/myurls:latest \
  -domain example.com \
  -port 8080 \
  -conn 192.168.1.100:6379 \
  -password REDIS_PASSWORD
```

请替换命令中的以下占位符：

- `example.com`：对外访问的服务域名。
- `192.168.1.100:6379`：Redis 实例地址与端口。
- `REDIS_PASSWORD`：Redis 访问密码。如无密码，传入空字符串 `""`。

### 使用二进制文件运行

1. 从 [GitHub Actions 构建产物](https://github.com/CareyWang/MyUrls/actions/workflows/go.yml) 下载对应平台的压缩包并解压。构建产物目前提供 Linux amd64、Linux arm64 和 Windows x64 版本；macOS 请参阅[本地编译](#本地编译)。以 Linux amd64 为例：

   ```bash
   tar -xzf myurls-linux-amd64.tar.gz
   cd myurls
   ```

2. 直接指定参数启动：

   ```bash
   ./myurls-linux-amd64 -domain example.com -port 8080 -conn 127.0.0.1:6379
   ```

3. 生产环境中，推荐使用进程守护工具（如 [PM2](https://pm2.keymetrics.io/) 或 systemd）进行托管：

   ```bash
   pm2 start ./myurls-linux-amd64 --name myurls -- -domain example.com -port 8080 -conn 127.0.0.1:6379
   ```

## 配置说明

MyUrls 支持通过命令行参数或环境变量进行配置。当两者同时存在时，**环境变量的优先级高于命令行参数**。

| 命令行参数 | 环境变量 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `-port` | `MYURLS_PORT` | `8080` | 服务监听的 HTTP 端口 |
| `-domain` | `MYURLS_DOMAIN` | `localhost:8080` | 生成短链接所使用的域名或主机头 |
| `-proto` | `MYURLS_PROTO` | `https` | 生成短链接所使用的 URL 协议（`http` 或 `https`） |
| `-conn` | `MYURLS_REDIS_CONN` | `localhost:6379` | Redis 服务器连接地址（`host:port`） |
| `-password` | `MYURLS_REDIS_PASSWORD` | `""` | Redis 认证密码 |
| `-h` | 无 | 无 | 打印命令行参数帮助信息并退出 |

## 接口说明

### 创建短链接

为指定的长链接创建短链接。

- **请求路径**：`/short`
- **请求方法**：`POST`
- **请求类型**：`application/json` 或 `application/x-www-form-urlencoded`

#### 请求参数

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `longUrl` | string | 是 | 长链接。必须是带主机名的 `http` 或 `https` URL，支持明文或 Base64 编码格式 |
| `shortKey` | string | 否 | 用户指定的短键，只能包含字母、数字、`-` 和 `_`。若留空，系统自动生成 7 位随机字符串 |

#### 请求示例

```bash
curl -X POST https://example.com/short \
  -H "Content-Type: application/json" \
  -d '{
    "longUrl": "https://www.example.com/very/long/path?param=value",
    "shortKey": "my-custom-key"
  }'
```

#### 响应说明

接口返回 HTTP 状态码 `200`，业务响应体为 JSON 结构。

- **成功响应**：

  ```json
  {
    "Code": 1,
    "ShortUrl": "https://example.com/my-custom-key"
  }
  ```

- **参数校验失败**（如缺少 `longUrl`）：

  ```json
  {
    "Code": 1001,
    "Message": "invalid parameters",
    "Data": null
  }
  ```

- **长链接无效**：

  ```json
  {
    "Code": 1001,
    "Message": "invalid long URL, only http and https URLs are supported",
    "Data": null
  }
  ```

- **短键无效**：

  ```json
  {
    "Code": 1001,
    "Message": "invalid short key, only letters, digits, hyphens, and underscores are allowed",
    "Data": null
  }
  ```

- **短键已被占用**（用户指定的 `shortKey` 已被一条有效映射占用）：

  ```json
  {
    "Code": 1001,
    "Message": "short key already exists, please use another one or leave it empty to generate automatically",
    "Data": null
  }
  ```

- **服务端错误**（Redis 存储失败、重试次数超限或随机数生成失败）：

  ```json
  {
    "Code": 1002,
    "Message": "failed to create short URL",
    "Data": null
  }
  ```

### 访问短链接

通过短键进行 HTTP 重定向跳转。

- **请求路径**：`/:shortKey`
- **请求方法**：`GET`

#### 响应说明

- **映射存在且有效**：返回 HTTP 状态码 `301 Moved Permanently`，并在 `Location` 响应头中携带长链接。
- **映射不存在或已过期**：返回 HTTP 状态码 `404 Not Found` 及 JSON 错误信息：

  ```json
  {
    "Code": 1002,
    "Message": "failed to get long URL, please check the short URL if exists or expired",
    "Data": null
  }
  ```

- **服务端错误**（Redis 读取失败）：返回 HTTP 状态码 `500 Internal Server Error` 及 JSON 错误信息：

  ```json
  {
    "Code": 1002,
    "Message": "failed to get long URL",
    "Data": null
  }
  ```

### Web 界面

访问服务根路径 `GET /` 可打开内置 Web 管理页面，静态文件在编译期已嵌入二进制文件内部，开箱即用。

## 核心机制

### 短键生成与冲突重试

- 当用户未指定 `shortKey` 时，系统基于 `crypto/rand` 随机源从 `[0-9a-zA-Z]` 字符集中生成 7 位长度的字符串。
- 系统使用 Redis `SETNX` 进行原子写入。若生成的新短键已存在，最多重试 3 次（连同初次生成共最多尝试 4 次）。
- 重试机制仅针对短键冲突触发；若在生成过程中发生底层随机数错误或 Redis 存储连接异常，系统立即中断并返回错误码 `1002`，不继续重试。

### 有效期与防覆盖规则

- 每条创建成功的短链接映射有效期为 365 天。
- 写入采用原子防覆盖机制。若短键已被占用，系统拒绝覆盖已有映射，且不会刷新或重置其剩余有效期。
- 为避免网络丢包导致写入命令重放引起假冲突，创建路径禁用了 Redis 命令盲重试策略。

## 构建与编译

### 本地编译

编译前请确保已安装 Go 1.27+：

```bash
# 安装依赖
make install

# 编译当前平台二进制文件（输出至 build/myurls）
make
```

### 跨平台编译

仓库内置了多平台交叉编译目标：

```bash
# 编译所有支持的平台
make all

# 单独编译各平台二进制文件
make linux          # Linux amd64
make aarch64        # Linux arm64
make darwin         # macOS amd64
make darwin-arm64   # macOS arm64 (Apple Silicon)
make windows        # Windows amd64 (输出 build/myurls-windows-x64)
```

> **说明**：首页静态资源位于 `public/` 目录下，由 `go:embed` 嵌入。如果修改了 `public/` 目录下的 HTML 或资源文件，必须重新编译方可生效。

## 日志与维护

### 内置日志轮转

MyUrls 内置基于 Lumberjack 的日志滚动机制。访问日志默认写入程序工作目录下的 `logs/access.log` 文件：

- 单个日志文件最大大小：50 MB
- 最多保留历史日志文件数：10 个
- 历史日志最长保留天数：7 天

### 使用 logrotate 维护（可选）

在基于 Linux 主机系统直接部署并由 systemd 托管的场景中，也可以配置外部 `logrotate` 进行日志管理。

创建 `/etc/logrotate.d/myurls` 文件，假定应用部署在 `/app`：

```logrotate
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
```

测试配置有效性（仅验证配置，不实际切割）：

```bash
logrotate -d /etc/logrotate.d/myurls
```

## 测试

项目测试套件基于 `miniredis` 实现了全隔离的内存 Redis 测试环境，执行测试无需连接外部 Redis 实例。

运行单元测试及并发竞态检测：

```bash
go test -race ./...
```

## 许可证

本项目采用 [MIT 许可证](LICENSE) 开源。

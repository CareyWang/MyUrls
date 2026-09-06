# MyUrls

基于 Go 1.26 与 Redis 实现的本地短链接服务，用于缩短 URL 与短链接还原。

## Table of Contents

- [Dependencies](#dependencies)
  - [Docker](#docker)
  - [Install](#install)
  - [Usage](#usage)
    - [短链接创建规则](#短链接创建规则)
    - [日志清理](#日志清理)
  - [测试](#测试)
  - [Maintainers](#maintainers)
  - [Contributing](#contributing)
  - [License](#license)

# Dependencies

本服务依赖于 Redis 提供长短链接映射关系存储，你需要本地安装 Redis 服务来保证短链接服务的正常运行。

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

```shell script
Usage of ./MyUrls:
  -conn string
        address of the redis server (default "localhost:6379")
  -domain string
        domain of the server (default "localhost:8080")
  -h    display help
  -password string
        password of the redis server
  -port string
        port to run the server on (default "8080")
  -proto string
        protocol of the server (default "https")
```

建议配合 [pm2](https://pm2.keymetrics.io/) 开启守护进程。

```shell script
pm2 start myurls --name myurls -- -domain example.com
```

### 短链接创建规则

每条新建映射的有效期为 365 天。创建时仅写入尚未使用的短键，不覆盖已有映射，也不延长已有映射的有效期。

- 用户指定短键时，发生冲突即返回原有的冲突提示。
- 未指定短键时，系统使用 `crypto/rand` 生成 7 位字母数字短键。发生冲突后最多重试 3 次，连同首次尝试共最多尝试 4 次。
- 自动生成短键的重试次数耗尽、随机数生成失败或存储操作失败时，返回服务器错误码 `1002` 和 `failed to create short URL`。随机数生成错误和存储错误均不触发短键生成重试。

`/short` 继续使用 HTTP 200 返回业务结果，成功响应保留 `Code: 1` 和 `ShortUrl` 字段。请求支持 JSON、表单和旧版 Base64 长链接。

创建路径不自动重发 Redis 写入命令。如果 Redis 已执行写入但响应丢失，系统返回存储错误，不将其误判为短键冲突，也不继续生成新短键。此时映射可能已经存在，错误响应不代表写入已回滚。其他 Redis 操作保留原有的命令重试策略。

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

## 测试

测试使用 [miniredis](https://github.com/alicebob/miniredis) 在进程内启动隔离的 Redis 替身，无需连接本机 Redis。测试覆盖并发创建、短键冲突重试、过期行为和请求响应兼容性。

运行测试及竞态检查：

```shell
go test -race ./...
```

## Maintainers

[@CareyWang](https://github.com/CareyWang)

## Contributing

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © 2024 CareyWang

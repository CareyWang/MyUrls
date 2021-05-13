# MyUrls

基于 golang1.15 与 Redis 实现的本地短链接服务，用于缩短请求链接与短链接还原。

## Table of Contents

- [Update](#update)
- [Dependencies](#dependencies)
- [Docker](#Docker)
- [Install](#install)
- [Usage](#usage)
- [API](#api)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [License](#license)

# Update

- 20200330
  集成前端至根路径，如: <http://127.0.0.1:8002/>。

  > 注：如需使用集成的前端，项目部署请 clone 仓库后自行编译，并在代码根目录启动服务。或者可 nginx 单独配置 root 至 public 目录的 index.html。


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
docker run -d --restart always --name myurls careywong/myurls:latest -domain example.com -port 8002 -conn 127.0.0.1:6379 -passwd '' -ttl 90
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
bash release.sh
```

## Usage

前往 [Release](https://github.com/CareyWang/MyUrls/releases) 下载对应平台可执行文件。

```shell script
./build/linux-amd64-myurls -h 

Usage of ./build/linux-amd64-myurls:
  -conn string
        Redis连接，格式: host:port (default "127.0.0.1:6379")
  -domain string
        短链接域名，必填项
  -passwd string
        Redis连接密码
  -port int
        服务端口 (default 8002)
  -ttl int
        短链接有效期，单位(天)，默认180天。 (default 180)
```

建议配合 [pm2](https://pm2.keymetrics.io/) 开启守护进程。

```shell script
pm2 start myurls --watch --name myurls -- -domain example.com
```

## API

[参考文档](https://myurls.mydoc.li)


## Maintainers

[@CareyWang](https://github.com/CareyWang)

## Contributing

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

MIT © 2020 CareyWang

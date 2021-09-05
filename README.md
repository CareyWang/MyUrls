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
## Cloud Deployment

现在你可以无需使用服务器，在线部署本项目，下面以 [railway.app](https://railway.app/) 为示例进行说明。若使用 Heroku 需要自行修改代码中的端口配置。 


### 参数说明

- `DOMAIN` - 短链接域名，必填项
- `CONN` - Redis连接，格式: host:port
- `PASSWD` - Redis连接密码
- `TTL` - 短链接有效期，单位(天)，默认180天。 (default 180)
- `PORT` - 端口，请勿修改

### 一键部署

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template?template=https%3A%2F%2Fgithub.com%2FCareyWang%2FMyUrls%2Ftree%2Fmaster%2Fonline&plugins=redis&envs=DOMAIN%2CCONN%2CPASSWD%2CTTL%2CPORT&optionalEnvs=PASSWD%2CTTL&DOMAINDesc=Short+link+domain+name%2C+required&CONNDesc=Redis+connection%2C+format%3A+host%3Aport&PASSWDDesc=Redis+connection+password&TTLDesc=The+validity+period+of+the+short+link+%28days%29%2C+default+180+days.+%28default+180%29&PORTDesc=DO+NOT+Change&PORTDefault=80)

### 其他说明

在线部署采用非本地Redis服务，可在railway.app部署项目后，添加Redius插件获取，或在 https://redislabs.com/redis-cloud 申请免费的Redis数据库。

若某些区域无法正常访问railway.app，可以通过 Cloudflare 配置CDN进行解决。

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

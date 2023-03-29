# MyUrls

基于 Go 1.20 与 Redis 实现的本地短链接服务，用于缩短请求链接与短链接还原。

## Table of Contents

- [MyUrls](#myurls)
  - [Table of Contents](#table-of-contents)
- [Update](#update)
- [Dependencies](#dependencies)
  - [Docker](#docker)
  - [Deploy Online](#deploy-online)
    - [Deploy on Railway](#deploy-on-railway)
      - [部署](#部署)
      - [添加域名](#添加域名)
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
## Deploy Online 

### Deploy on Railway

#### 部署

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/new/template?template=https%3A%2F%2Fgithub.com%2Fpzcn%2FMyurls-Railway&plugins=redis&envs=ENV_DOMAIN%2CENV_TTL%2CPORT&ENV_DOMAINDesc=Your+domain.&ENV_TTLDesc=Short+link+validity+period+%28day%29&PORTDesc=DO+NOT+CHANGE&ENV_TTLDefault=180&PORTDefault=80)

通过上方链接一键部署到Railway，并填入以下参数

参数说明：

- `DOMAIN` - 短链接域名，必填项，不需要添加https:// (如 abc.com)
- `TTL` - 短链接有效期，单位(天)，默认180天 (default 180)
- `PORT` - 端口，保持80，请勿修改

#### 添加域名

在Cloudflare中添加域名，并配置SSL/TLS为完全及以上，并在Railway中接入该域名，参考[官方文档](https://docs.railway.app/deploy/exposing-your-app#lets-encrypt-ssl-certificates)。

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

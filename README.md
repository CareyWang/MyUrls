# bitly

基于 golang1.13 与 Redis 实现的本地短链接服务，用于缩短请求链接与短链接还原。

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

TODO

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

前往 [Release](https://github.com/CareyWang/MyUrls/releases) 下载对应平台可执行文件。

```shell script
./build/myurls -h 

Usage of ./build/myurls.service:
  -domain string
        短链接域名 (default "s.wcc.best")
  -port int
        服务端口 (default 8002)
  -ttl int
        短链接有效期，单位(天)，默认90天。 (default 90)
```

建议配合 [pm2](https://pm2.keymetrics.io/) 开启守护进程。

```shell script
pm2 start myurls.service --watch --name myurls -- -domain example.com
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

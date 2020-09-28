# MyUrls

![GitHub release (latest by date)](https://img.shields.io/github/v/release/careywang/myurls)
![golang version](https://img.shields.io/badge/Golang-1.13-brightgreen)
![GitHub commits since latest release (by date)](https://img.shields.io/github/commits-since/careywang/myurls/latest/master)
![GitHub last commit](https://img.shields.io/github/last-commit/careywang/myurls)
![GitHub contributors](https://img.shields.io/github/contributors/careywang/myurls)

基于 golang1.13 与 Redis 实现的本地短链接服务，用于缩短请求链接与短链接还原。

[English README](/README.md)

## 目录

- [更新](#更新)
- [依赖](#依赖)
- [Docker](#Docker)
- [安装](#安装)
- [使用](#使用)
- [API](#api)
- [维护者](#维护者)
- [贡献](#贡献)
- [License](#license)

# 更新

- 20200928

  编译arm64架构二进制文件并加入release，现在你可以在树莓派以及其他arm64架构的平台上使用它。

- 20200330

  集成前端至根路径，如: <http://127.0.0.1:8002/>。

  > 注：如需使用集成的前端，项目部署请 clone 仓库后自行编译，并在代码根目录启动服务。或者可 nginx 单独配置 root 至 public 目录的 index.html。


# 依赖

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

## 安装

安装项目依赖

```shell script
make install
```

生成可执行文件，目录位于 build/ ，默认当前平台。

```shell script
make
```

其他平台交叉编译

```shell script
# Run these command no matter what platform you are using
go env -w GO111MODULE="on" && go env -w GOPROXY="https://goproxy.cn,direct"
go mod tidy 

# Cross-compiling by change the value of "GOOS" and "GOARCH"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myurls main.go 
```

## 使用

前往 [Release](https://github.com/CareyWang/MyUrls/releases) 下载对应平台可执行文件。

```shell script
./build/linux-amd64-myurls.service -h 

Usage of ./build/linux-amd64-myurls.service:
  -conn string
        Redis连接，格式: host:port (default "127.0.0.1:6379")
  -domain string
        短链接域名，必填项
  -passwd string
        Redis连接密码
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


## 维护者

[@CareyWang](https://github.com/CareyWang)

## 贡献

接受PR

小提示：如果编辑自述文件，请遵循[standard-readme]（https://github.com/RichardLitt/standard-readme)规范。

💖 特别感谢 **Fanyang Meng** [@MFYDev](https://github.com/MFYDev)的贡献。

## License

MIT © 2020 CareyWang

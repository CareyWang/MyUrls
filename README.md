# MyUrls

![GitHub release (latest by date)](https://img.shields.io/github/v/release/careywang/myurls)
![golang version](https://img.shields.io/badge/Golang-1.13-brightgreen)
![GitHub commits since latest release (by date)](https://img.shields.io/github/commits-since/careywang/myurls/latest/master)
![GitHub last commit](https://img.shields.io/github/last-commit/careywang/myurls)
![GitHub contributors](https://img.shields.io/github/contributors/careywang/myurls)

Local short link service based on Golang 1.13 and Redis for shortening request link and short link restoration.

[中文文档](/README-CN.md)

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

- 20200928

  Compile the ARM64 architecture binary and add it to the Release, which you can now use on raspberry PI and other ARM64 architecture platforms.
  
- 20200330

  Integrate the front end to the root path, such as: <http://127.0.0.1:8002/>。

  > Note: To use an integrated front end, clone the repository for project deployment and start the service in the root directory or nginx can configure index.html from root to public separately


# Dependencies

This service relies on Redis to provide long and short link mapping relational storage. You need to install the Redis service locally to keep the short link service running.

```shell script
sudo apt-get update

# Install Redis
sudo add-apt-repository ppa:chris-lea/redis-server -y 
sudo apt-get update 
sudo apt-get install redis-server -y 
```

## Docker 

Now you can use docker or [docker-compose](https://docs.docker.com/compose/install/) to deploy this project without installing other services. 

Note: Please modify the parameters in .env by yourself.

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

Installation project dependencies

```shell script
make install
```

Generate executable files, the directory is located in build/. The current platform is the default. For other platforms, cross-compiling will be covered in the following part.

```shell script
make
```
Cross-compiling

```shell script
# Run these command no matter what platform you are using
go env -w GO111MODULE="on" && go env -w GOPROXY="https://goproxy.cn,direct"
go mod tidy 

# Cross-compiling by change the value of "GOOS" and "GOARCH"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o myurls main.go 
```

## Usage

Go to [Release](https://github.com/CareyWang/MyUrls/releases) to download the corresponding platform executable file.

```shell script
./build/linux-amd64-myurls.service -h 

Usage of ./build/linux-amd64-myurls.service:
  -conn string
        Redis connection, format: host:port (default "127.0.0.1:6379")
  -domain string
        Short link domain name, required
  -passwd string
        Redis connection password
  -port int
        Service port (default 8002)
  -ttl int
        Short link validity, unit (days) (default 90)
```

It is recommended to start the daemon with [pm2](https://pm2.keymetrics.io/).

```shell script
pm2 start myurls.service --watch --name myurls -- -domain example.com
```

## API

[Reference](https://myurls.mydoc.li)


## Maintainers

[@CareyWang](https://github.com/CareyWang)

## Contributing

PRs accepted.

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

💖 Special Thanks to **FanyangMeng** [@MFYDev](https://github.com/MFYDev) for his contributing.

## License

MIT © 2020 CareyWang

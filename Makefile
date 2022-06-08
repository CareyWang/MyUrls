BINARY_DEFAULT="build/myurls"
BINARY_LINUX="build/myurls-linux-amd64"
BINARY_DARWIN="build/myurls-darwin-amd64"
BINARY_WINDOWS="build/myurls-windows-x64"
BINARY_ARM64="build/myurls-arm64"

GOFILES="main.go"
VERSION=1.0.0
BUILD=`date +%FT%T%z`

default:
	@echo ${BINARY_DEFAULT}
	@go build -o ${BINARY_DEFAULT} ${GOFILES}

all:
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY_LINUX} ${GOFILES}
	@echo ${BINARY_DARWIN}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${BINARY_DARWIN} ${GOFILES}
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ${BINARY_WINDOWS} ${GOFILES}
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${BINARY_ARM64} ${GOFILES}

linux:
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY_LINUX} ${GOFILES}

darwin:
	@echo ${BINARY_DARWIN}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${BINARY_DARWIN} ${GOFILES}

windows:
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ${BINARY_WINDOWS} ${GOFILES}

aarch64:
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${BINARY_ARM64} ${GOFILES}

install:
	@go mod tidy

fmt:
	@go fmt ./...

clean:
	@if [ -f ${BINARY_DEFAULT} ] ; then rm ${BINARY_DEFAULT} ; fi
	@if [ -f ${BINARY_LINUX} ] ; then rm ${BINARY_LINUX} ; fi
	@if [ -f ${BINARY_DARWIN} ] ; then rm ${BINARY_DARWIN} ; fi
	@if [ -f ${BINARY_WINDOWS} ] ; then rm ${BINARY_WINDOWS} ; fi
	@if [ -f ${BINARY_ARM64} ] ; then rm ${BINARY_ARM64} ; fi

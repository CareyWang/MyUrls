BINARY_DEFAULT="build/myurls.service"
BINARY_LINUX="build/linux-amd64-myurls.service"
BINARY_DARWIN="build/darwin-amd64-myurls.service"
BINARY_WINDOWS="build/windows-amd64-myurls.service"
BINARY_ARRCH64="build/arrch64-myurls.service"

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
	@echo ${BINARY_ARRCH64}
	@CGO_ENABLED=0 GOOS=windows GOARCH=aarch64 go build -o ${BINARY_ARRCH64} ${GOFILES}

install:
	@go mod tidy

fmt:
	@go fmt ./...

clean:
	@if [ -f ${BINARY_DEFAULT} ] ; then rm ${BINARY_DEFAULT} ; fi
	@if [ -f ${BINARY_LINUX} ] ; then rm ${BINARY_LINUX} ; fi
	@if [ -f ${BINARY_DARWIN} ] ; then rm ${BINARY_DARWIN} ; fi
	@if [ -f ${BINARY_WINDOWS} ] ; then rm ${BINARY_WINDOWS} ; fi
	@if [ -f ${BINARY_ARRCH64} ] ; then rm ${BINARY_ARRCH64} ; fi

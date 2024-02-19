BINARY_DEFAULT="build/myurls"
BINARY_LINUX="build/myurls-linux-amd64"
BINARY_DARWIN="build/myurls-darwin-amd64"
BINARY_DARWIN_ARM64="build/myurls-darwin-arm64"
BINARY_WINDOWS="build/myurls-windows-x64"
BINARY_ARM64="build/myurls-linux-arm64"

VERSION=1.0.0
BUILD=`date +%FT%T%z`

default:
	@echo ${BINARY_DEFAULT}
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o ${BINARY_DEFAULT}

all:
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_LINUX}
	# @echo ${BINARY_DARWIN}
	# @CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_DARWIN}
	# @echo ${BINARY_DARWIN_ARM64}
	# @CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_DARWIN_ARM64}
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_WINDOWS}
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_ARM64}

linux:
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_LINUX}

darwin:
	@echo ${BINARY_DARWIN}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_DARWIN}

windows:
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_WINDOWS}

aarch64:
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_ARM64}

install:
	@go mod tidy

fmt:
	@go fmt ./...

clean:
	@if [ -f ${BINARY_DEFAULT} ] ; then rm ${BINARY_DEFAULT} ; fi
	@if [ -f ${BINARY_LINUX} ] ; then rm ${BINARY_LINUX} ; fi
	@if [ -f ${BINARY_DARWIN} ] ; then rm ${BINARY_DARWIN} ; fi
	@if [ -f ${BINARY_DARWIN_ARM64} ] ; then rm ${BINARY_DARWIN_ARM64} ; fi
	@if [ -f ${BINARY_WINDOWS} ] ; then rm ${BINARY_WINDOWS} ; fi
	@if [ -f ${BINARY_ARM64} ] ; then rm ${BINARY_ARM64} ; fi

BINARY_DEFAULT="output/myurls"
BINARY_LINUX="output/myurls-linux-amd64"
BINARY_DARWIN="output/myurls-darwin-amd64"
BINARY_DARWIN_ARM64="output/myurls-darwin-arm64"
BINARY_WINDOWS="output/myurls-windows-x64.exe"
BINARY_ARM64="output/myurls-linux-arm64"

SYNC_DATA_BINARY="output/sync_data"

VERSION=1.0.0
BUILD=`date +%FT%T%z`

default:
	@echo ${BINARY_DEFAULT}
	@mkdir -p output
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o ${BINARY_DEFAULT} ./cmd/myurls
	@cp -r conf output/
	@cp -r web output/

sync_data:
	@echo ${SYNC_DATA_BINARY}
	@mkdir -p output
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o ${SYNC_DATA_BINARY} ./cmd/sync_data
	@cp -r conf output/

all:
	@mkdir -p output
	@cp -r conf output/
	@cp -r web output/
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_LINUX} ./cmd/myurls
	@echo ${BINARY_DARWIN}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_DARWIN} ./cmd/myurls
	@echo ${BINARY_DARWIN_ARM64}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_DARWIN_ARM64} ./cmd/myurls
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_WINDOWS} ./cmd/myurls
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_ARM64} ./cmd/myurls

linux:
	@mkdir -p output
	@cp -r conf output/
	@cp -r web output/
	@echo ${BINARY_LINUX}
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_LINUX} ./cmd/myurls

darwin:
	@mkdir -p output
	@cp -r conf output/
	@cp -r web output/
	@echo ${BINARY_DARWIN}
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_DARWIN} ./cmd/myurls

windows:
	@mkdir -p output
	@cp -r conf output/
	@cp -r web output/
	@echo ${BINARY_WINDOWS}
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_WINDOWS} ./cmd/myurls

aarch64:
	@mkdir -p output
	@cp -r conf output/
	@cp -r web output/
	@echo ${BINARY_ARM64}
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o ${BINARY_ARM64} ./cmd/myurls

install:
	@go mod tidy

fmt:
	@go fmt ./...

clean:
	@rm -rf output/

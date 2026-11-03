BINARY_DEFAULT=build/myurls
BINARY_LINUX=build/myurls-linux-amd64
BINARY_DARWIN=build/myurls-darwin-amd64
BINARY_DARWIN_ARM64=build/myurls-darwin-arm64
BINARY_WINDOWS=build/myurls-windows-x64
BINARY_ARM64=build/myurls-linux-arm64

# cross_build builds for GOOS $(1) and GOARCH $(2) into $(3).
define cross_build
	@echo $(3)
	@CGO_ENABLED=0 GOOS=$(1) GOARCH=$(2) go build -ldflags="-s -w" -o $(3)
endef

default:
	@echo ${BINARY_DEFAULT}
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o ${BINARY_DEFAULT}

all: linux darwin darwin-arm64 windows aarch64

linux:
	$(call cross_build,linux,amd64,${BINARY_LINUX})

darwin:
	$(call cross_build,darwin,amd64,${BINARY_DARWIN})

darwin-arm64:
	$(call cross_build,darwin,arm64,${BINARY_DARWIN_ARM64})

windows:
	$(call cross_build,windows,amd64,${BINARY_WINDOWS})

aarch64:
	$(call cross_build,linux,arm64,${BINARY_ARM64})

install:
	@go mod tidy

fmt:
	@go fmt ./...

clean:
	@rm -f ${BINARY_DEFAULT} ${BINARY_LINUX} ${BINARY_DARWIN} ${BINARY_DARWIN_ARM64} ${BINARY_WINDOWS} ${BINARY_ARM64}

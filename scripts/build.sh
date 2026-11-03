#!/bin/bash

set -e

platform=${1:?usage: scripts/build.sh <linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-x64>}

case "$platform" in
	linux-amd64) target=linux ;;
	linux-arm64) target=aarch64 ;;
	darwin-amd64) target=darwin ;;
	darwin-arm64) target=darwin-arm64 ;;
	windows-x64) target=windows ;;
	*) echo "unknown platform: $platform" >&2; exit 1 ;;
esac

make install
make "$target"

mkdir -p myurls

cp "build/myurls-${platform}" myurls/
tar -czvf "myurls-${platform}.tar.gz" myurls
mv "myurls-${platform}.tar.gz" build/
rm -rf myurls

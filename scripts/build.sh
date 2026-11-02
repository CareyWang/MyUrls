#!/bin/bash

set -e

platform=${1:?usage: scripts/build.sh <linux-amd64|linux-arm64|darwin-amd64|darwin-arm64|windows-x64>}

make install
make all

mkdir -p myurls

cp "build/myurls-${platform}" myurls/
tar -czvf "myurls-${platform}.tar.gz" myurls
mv "myurls-${platform}.tar.gz" build/
rm -rf myurls

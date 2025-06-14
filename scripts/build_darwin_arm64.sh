#!/bin/bash

# 确保 build 目录存在
mkdir -p build

make install
make all

mkdir -p myurls
cp -r web myurls/

# darwin-arm64
cp build/myurls-darwin-arm64 myurls/
tar -czvf myurls-darwin-arm64.tar.gz myurls
mv myurls-darwin-arm64.tar.gz build/
rm -rf myurls 
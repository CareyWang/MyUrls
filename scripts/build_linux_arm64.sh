#!/bin/bash

# 确保 build 目录存在
mkdir -p build

make install
make all

mkdir -p myurls
cp -r web myurls/

# linux-arm64
cp build/myurls-linux-arm64 myurls/
tar -czvf myurls-linux-arm64.tar.gz myurls
mv myurls-linux-arm64.tar.gz build/
rm -rf myurls

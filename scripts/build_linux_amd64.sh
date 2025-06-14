#!/bin/bash

# 确保 build 目录存在
mkdir -p build

make install
make all

mkdir -p myurls
cp -r web myurls/

# linux-amd64
cp build/myurls-linux-amd64 myurls/
tar -czvf myurls-linux-amd64.tar.gz myurls
mv myurls-linux-amd64.tar.gz build/
rm -rf myurls

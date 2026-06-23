#!/bin/bash

# 确保 output 目录存在
mkdir -p output

make install
make all

mkdir -p myurls
cp -r web conf myurls/

# linux-arm64
cp output/myurls-linux-arm64 myurls/
tar -czvf myurls-linux-arm64.tar.gz myurls
mv myurls-linux-arm64.tar.gz output/
rm -rf myurls

#!/bin/bash

# 确保 output 目录存在
mkdir -p output

make install
make all

mkdir -p myurls
cp -r web conf myurls/

# darwin-amd64
cp output/myurls-darwin-amd64 myurls/
tar -czvf myurls-darwin-amd64.tar.gz myurls
mv myurls-darwin-amd64.tar.gz output/
rm -rf myurls

#!/bin/bash

# 确保 build 目录存在  
mkdir -p build

make install
make all

mkdir -p myurls
cp -r web myurls/

# windows-x64
cp build/myurls-windows-x64.exe myurls/
tar -czvf myurls-windows-x64.tar.gz myurls
mv myurls-windows-x64.tar.gz build/
rm -rf myurls

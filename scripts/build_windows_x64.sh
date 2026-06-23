#!/bin/bash

# 确保 output 目录存在  
mkdir -p output

make install
make all

mkdir -p myurls
cp -r web conf myurls/

# windows-x64
cp output/myurls-windows-x64.exe myurls/
tar -czvf myurls-windows-x64.tar.gz myurls
mv myurls-windows-x64.tar.gz output/
rm -rf myurls

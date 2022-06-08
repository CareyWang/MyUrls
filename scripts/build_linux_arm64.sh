#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# linux-arm64
cp build/myurls-linux-arm64 myurls/
tar -czvf myurls-linux-arm64.tar.gz myurls
mv myurls-linux-arm64.tar.gz build/
rm -rf myurls

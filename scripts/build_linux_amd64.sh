#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# linux-amd64
cp build/myurls-linux-amd64 myurls/
tar -czvf myurls-linux-amd64.tar.gz myurls
mv myurls-linux-amd64.tar.gz build/
rm -rf myurls

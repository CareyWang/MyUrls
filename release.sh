#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# linux-amd64
cp build/myurls-linux-amd64 myurls/
tar -czvf myurls-linux-amd64.tar.gz myurls
mv myurls-linux-amd64.tar.gz build/
rm build/myurls-linux-amd64
rm -rf myurls/*

# arrch64
cp build/myurls-arrch64 myurls/
tar -czvf myurls-arrch64.tar.gz myurls
mv myurls-arrch64.tar.gz build/
rm build/myurls-arrch64
rm -rf myurls/*

# darwin-amd64
cp build/myurls-darwin-amd64 myurls/
tar -czvf myurls-darwin-amd64.tar.gz myurls
mv myurls-darwin-amd64.tar.gz build/
rm build/myurls-darwin-amd64
rm -rf myurls/*

# windows-x64
cp build/myurls-windows-x64 myurls/
tar -czvf myurls-windows-x64.tar.gz myurls
mv myurls-windows-x64.tar.gz build/
rm build/myurls-windows-x64
rm -rf myurls/*

rm -rf myurls

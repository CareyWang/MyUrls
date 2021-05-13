#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# linux-amd64
cp build/linux-amd64-myurls myurls/
tar -czvf linux-amd64-myurls.tar.gz myurls
mv linux-amd64-myurls.tar.gz build/
rm build/linux-amd64-myurls
rm -rf myurls/*

# arrch64
cp build/arrch64-myurls myurls/
tar -czvf arrch64-myurls.tar.gz myurls
mv arrch64-myurls.tar.gz build/
rm build/arrch64-myurls
rm -rf myurls/*

# darwin-amd64
cp build/darwin-amd64-myurls myurls/
tar -czvf darwin-amd64-myurls.tar.gz myurls
mv darwin-amd64-myurls.tar.gz build/
rm build/darwin-amd64-myurls
rm -rf myurls/*

# windows-x64
cp build/windows-x64-myurls myurls/
tar -czvf windows-x64-myurls.tar.gz myurls
mv windows-x64-myurls.tar.gz build/
rm build/windows-x64-myurls
rm -rf myurls/*

rm -rf myurls

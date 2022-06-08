#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# windows-x64
cp build/myurls-windows-x64 myurls/
tar -czvf myurls-windows-x64.tar.gz myurls
mv myurls-windows-x64.tar.gz build/
rm -rf myurls

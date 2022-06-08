#!/bin/bash

make install
make all

mkdir -p myurls
cp -r public myurls/

# darwin-amd64
cp build/myurls-darwin-amd64 myurls/
tar -czvf myurls-darwin-amd64.tar.gz myurls
mv myurls-darwin-amd64.tar.gz build/
rm -rf myurls

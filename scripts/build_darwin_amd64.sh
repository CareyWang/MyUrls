#!/bin/bash

set -e

make install
make all

mkdir -p myurls

# darwin-amd64
cp build/myurls-darwin-amd64 myurls/
tar -czvf myurls-darwin-amd64.tar.gz myurls
mv myurls-darwin-amd64.tar.gz build/
rm -rf myurls

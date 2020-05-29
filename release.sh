#!/bin/bash

make all
mkdir -p myurls
cp -r public myurls/

cp build/linux-amd64-myurls.service myurls/

tar -czvf linux-amd64.tar.gz myurls
mv linux-amd64.tar.gz build/

rm -rf myurls

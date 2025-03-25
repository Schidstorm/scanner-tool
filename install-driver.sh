#!/bin/bash

set -euo pipefail

# download source from http://download.ebz.epson.net/dsc/search/01/search/?OSC=LX
epsonscan2FileUrl="https://download3.ebz.epson.net/dsc/f/03/00/16/14/37/7577ee65efdad48ee2d2f38d9eda75418e490552/epsonscan2-6.7.70.0-1.src.tar.gz"

WORK_DIR=$(mktemp -d $(dirname $0)/tmp.XXXXXX)
cleanup() {
    echo "cleanup"
    rm -rf "$WORK_DIR"
}
trap cleanup EXIT

# download
epsonscan2FileName=$(basename $epsonscan2FileUrl)
wget -O $WORK_DIR/$epsonscan2FileName "$epsonscan2FileUrl"

# check sha256
echo "e141e66e4cd74c06eef0baa163f4cc498c3be4a5db82851c914a1b7f2c50967e  $WORK_DIR/$epsonscan2FileName" | sha256sum --check

# extract
tar -xzf $WORK_DIR/$epsonscan2FileName -C $WORK_DIR
srcPath=$(find $WORK_DIR -type d -name 'epsonscan2-*')
pushd $srcPath

# install epsonscan2 dependencies
sudo apt-get install -y \
    libgtk2.0-dev \
    libusb-1.0-0-dev \
    libjpeg-dev \
    libtiff-dev \
    libpng-dev \
    libavahi-client-dev \
    libavahi-glib-dev \
    libboost-dev \
    qtbase5-dev \
    build-essential \
    cmake

# cleanup CMakeCache.txt
rm -f CMakeCache.txt

# build
mkdir -p build
pushd build
cmake ..
make
sudo make install

popd
popd


# install scanner-tool dependencies
sudo apt-get install -y tesseract-ocr-deu tesseract-ocr-eng tesseract-ocr
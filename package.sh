#!/bin/bash
#
# Simple script for creating releases and optionally uploading to GitHub
#
# To determine the release ID, execute this command:
#
#   curl https://api.github.com/repos/ekanite/ekanite/releases

if [ $# -lt 1 ]; then
    echo "$0 <version> [release_id api_token]"
    exit 1
fi

REPO_URL="https://github.com/ekanite/ekanite"

VERSION=$1
RELEASE_ID=$2
API_TOKEN=$3

tmp_build=`mktemp -d`
tmp_pkg=`mktemp -d`

kernel=`uname -s`
machine=`uname -m`
if [ "$machine" == "x86_64" ]; then
    machine="amd64"
fi
branch=`git rev-parse --abbrev-ref HEAD`
commit=`git rev-parse HEAD`
kernel=`uname -s`

mkdir -p $tmp_build/src/github.com/ekanite
export GOPATH=$tmp_build
cd $tmp_build/src/github.com/ekanite
git clone $REPO_URL
cd ekanite
go get -d ./...
go install -ldflags="-X main.version=$VERSION -X main.branch=$branch -X main.commit=$commit" ./...

release=`echo ekanited-$VERSION-$kernel-$machine | tr '[:upper:]' '[:lower:]'`
release_pkg=${release}.tar.gz
mkdir $tmp_pkg/$release
cp $GOPATH/bin/ekanited $tmp_pkg/$release
cd $tmp_pkg
tar cvfz $release_pkg $release

if [ -z "$API_TOKEN" ]; then
    exit 0
fi

upload_url="https://uploads.github.com/repos/ekanite/ekanite/releases/$RELEASE_ID/assets"
curl -v -H "Content-type: application/octet-stream" -H "Authorization: token $API_TOKEN" -XPOST $upload_url?name=$release_pkg --data-binary @$release_pkg

#!/bin/bash
set -e

CNOC="\033[0m"
CAOK="\033[32;01m"
CERR="\033[31;01m"
CWRN="\033[33;01m"

echo -e "Running in ${CAOK}$PWD $CNOC"
REPO=$(basename $PWD)
if [ "$REPO" != "gin-repo" ]; then
    echo -e "${CERR}* Not in gin-repo *{CNOC}"
    exit 1
fi

echo "Pulling latest changes"
BRANCH=$(git rev-parse --abbrev-ref HEAD)

if [ "$BRANCH" != "master" ]; then
    echo -e "${CERR}* Not on master${CNOC} [${CWRN}$BRANCH${CNOC}]"
    exit 1
fi

git pull origin master

echo "Processing dependencies"
ALLDEPS=$(go list -f '{{ join .Deps "\n" }}' ./... | sort -u | grep -v -e "github.com/G-Node/gin-repo" -e "golang.org/x/");
STDDEPS=$(go list std);
EXTDEPS=$(comm -23 <(echo "$ALLDEPS") <(echo "$STDDEPS"))

for dep in "$EXTDEPS"; do
    go get -v $dep
done

echo "Building & installing"
go install -v ./...

SRC="$GOPATH/bin/gin-shell"
DST="/usr/bin/gin-shell"
echo "Copying gin-shell"
echo -e "\t $SRC -> $DST"
sudo cp "$SRC" "$DST"

echo "Restarting gin-repod"
sudo systemctl restart ginrepo.service
sudo systemctl status ginrepo.service

echo "${CAOK}Done{CNOC}."

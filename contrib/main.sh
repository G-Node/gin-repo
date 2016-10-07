#!/bin/bash

echo "Starting gin-repo service"
echo " GRD_GENDATA=${GRD_GENDATA}"

if [ ! -z ${GRD_GENDATA+x} ]; then
    git config --global user.name "gin repo docker"
    git config --global user.email "gin-repo@g-node.org"
    DATA=${GRD_DATAFILE:-$GOPATH/src/github.com/G-Node/gin-repo/contrib/data.yml}
    echo "Generating data [$DATA]"
    $GOPATH/src/github.com/G-Node/gin-repo/contrib/mkdata.py $DATA
fi

supervisord -c/etc/supervisord.conf

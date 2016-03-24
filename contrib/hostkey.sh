#!/bin/bash

HOSTKEY="$PWD/ssh_host_rsa_key"

if [ ! -e "$HOSTKEY" ]; then
    ssh-keygen -b 4096 -t rsa -f "$HOSTKEY" -P ""
fi

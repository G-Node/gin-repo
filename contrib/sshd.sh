#!/bin/bash

SSHDCFG="$PWD/sshd.cfg"
BINDIR="$GOPATH/bin"
BINARY=`realpath $BINDIR/gin-repo`

if [ ! -x "$BINARY" ]; then
   echo "$BINARY does not exist (or is not executable)"
   exit -1
fi

HOSTKEY="$PWD/ssh_host_rsa_key"

if [ ! -e "$HOSTKEY" ]; then
    ssh-keygen -t rsa -f "$HOSTKEY" -P ""
fi

SSHD=`which sshd`

PORT="22222"
USER=`whoami`

cat << EOF > "$SSHDCFG"
Port $PORT
AddressFamily inet
HostKey $HOSTKEY
UsePrivilegeSeparation no
AuthorizedKeysCommand /gin-repo keys sshd %f
AuthorizedKeysCommandUser gicmo
UsePam no
PidFile $PWD/sshd.pid
EOF

AUTHKEYS="$PWD/ssh_authorized_keys"
echo "command .. $BINARY"
echo "sshd ..... $SSHD"
echo "pwd ...... $PWD"
echo "cfg ...... $SSHDCFG"
echo "host key . $HOSTKEY"
echo "port ..... $port"

"$SSHD" -De -f "$SSHDCFG"

#!/bin/bash
set -e
#Use with GIT_SSH=gin_ss.sh git clone git@github.com:/user/repo

ssh -i "$GIN_SSHID" -p "$GIN_SSHPORT" -oStrictHostKeyChecking=no $1 $2

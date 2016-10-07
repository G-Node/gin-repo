#!/bin/bash

#Ensure we have the right permissions on /data
find . -perm 0444 -exec chmod 0666 {} \; -exec chown git:git {} \; -exec chmod 0444 {} \;
chown -Rf git:git /data
supervisord -c/etc/supervisord.conf

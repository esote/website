#!/bin/sh

set -e

# Run builder
su -l -s /bin/ksh builder -c "./builder.sh"

builder=/home/builder
bin=$builder/go/bin
src=$builder/go/src/github.com/esote

gitweb_conf=$src/website/config/gitweb.conf
redirect_conf=$src/website/config/redirect.conf

# Kill existing servers.
pkill -9 web-srv fmtc indent.out gitweb redirect.out web-proxy || true

# Deploy servers.
su -l -s /bin/ksh run-web -c "cd $src/website; $bin/web-srv >> log 2>&1 &"
su -l -s /bin/ksh run-fmtc -c "$bin/fmtc >> log 2>&1 &"
su -l -s /bin/ksh run-chat -c "$bin/chat >> log 2>&1 &"
su -l -s /bin/ksh run-gitweb -c "$bin/gitweb $gitweb_conf >> log 2>&1 &"
su -l -s /bin/ksh run-redirect -c "$builder/redirect/redirect.out $redirect_conf >> log 2>&1 &"

su -l -s /bin/ksh run-proxy

pkill -9 proxy.sh

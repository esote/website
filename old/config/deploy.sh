#!/bin/sh

set -e
set -o xtrace

# Kill existing servers.
pkill -9 web-srv fmtc indent.out gitweb redirect.out web-proxy || true

# Run builder
su -l -s /bin/ksh builder -c "./builder.sh"

builder=/home/builder
bin=$builder/go/bin

gitweb_conf=$builder/website/config/gitweb.conf
redirect_conf=$builder/website/config/redirect.conf

# Deploy servers.
su -l -s /bin/ksh run-web -c "cd $builder/website; $bin/web-srv >> ~/log 2>&1 &"
su -l -s /bin/ksh run-fmtc -c "cd $builder/indent; $bin/fmtc >> ~/log 2>&1 &"
su -l -s /bin/ksh run-chat -c "$bin/chat >> ~/log 2>&1 &"
su -l -s /bin/ksh run-gitweb -c "$bin/gitweb $gitweb_conf >> ~/log 2>&1 &"
su -l -s /bin/ksh run-redirect -c "$builder/redirect/redirect.out $redirect_conf >> ~/log 2>&1 &"

su -l -s /bin/ksh run-proxy

pkill -9 proxy.sh

#!/bin/sh

set -e

# Run builder.sh script
su -l -s /bin/ksh builder -c "./builder.sh"

builder=/home/builder
bin=$builder/go/bin
src=$builder/go/src/github.com/esote

# Kill existing servers.
pkill -9 web-srv fmtc indent.out gitweb redirect.out web-proxy || true

# Deploy servers.
su -l -s /bin/ksh run-web -c "cd $src/website; $bin/web-srv &" &
su -l -s /bin/ksh run-fmtc -c "$bin/fmtc & " &
su -l -s /bin/ksh run-chat -c "$bin/chat &" &
su -l -s /bin/ksh run-gitweb -c "cd $builder; $bin/gitweb &" &
su -l -s /bin/ksh run-redirect -c "$builder/redirect/redirect.out &" &

su -l -s /bin/ksh run-proxy

pkill -9 proxy.sh

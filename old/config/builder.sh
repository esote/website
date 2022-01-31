#!/bin/sh

set -e
set -o xtrace

rm -rf repos/ redirect/ indent/ website/

website() {
	go get -u github.com/esote/website/cmd/web-gen
	go get -u github.com/esote/website/cmd/web-srv
	go get -u github.com/esote/website/cmd/web-proxy
	git clone https://github.com/esote/website
}

redirect() {
	git clone https://github.com/esote/redirect
	make -C redirect/
}

indent() {
	git clone https://github.com/esote/indent
	make -C indent/
}

website &
redirect &
indent &

go get -u github.com/esote/fmtc &
go get -u github.com/esote/chat &
go get -u github.com/esote/gitweb &

# Gitweb repositories
mkdir repos
git -C repos/ clone https://github.com/esote/mcc &
git -C repos/ clone https://github.com/esote/opal &
git -C repos/ clone https://github.com/esote/pof &

wait

echo "Awaiting POF"

cd ~/website; ~/go/bin/web-gen

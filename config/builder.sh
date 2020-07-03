#!/bin/sh

set -e
set -o xtrace

rm -rf go/ repos/ redirect/ indent/

website() {
	go get github.com/esote/website/cmd/web-gen
	go get github.com/esote/website/cmd/web-srv
	go get github.com/esote/website/cmd/web-proxy
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

go get github.com/esote/fmtc &
go get github.com/esote/chat &
go get github.com/esote/gitweb &

# Gitweb repositories
mkdir repos
git -C repos/ clone https://github.com/esote/mcc &
git -C repos/ clone https://github.com/esote/opal &
git -C repos/ clone https://github.com/esote/pof &

wait

echo "Awaiting POF"

cd ~/go/src/github.com/esote/website; ~/go/bin/web-gen

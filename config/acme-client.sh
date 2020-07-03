#!/bin/sh

# Run every ten days at midnight to check for cert renewal.
echo "Pausing web-proxy"
pkill -SIGUSR1 web-proxy

echo "Starting httpd"
rcctl -f start httpd

echo "Running acme-client"
acme-client esote.net

echo "Stopping httpd"
rcctl stop httpd

echo "Resuming web-proxy"
pkill -SIGUSR1 web-proxy

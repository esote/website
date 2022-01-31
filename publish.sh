git -C ~/website pull
rm -rf /var/www/esote.net
cp -r ~/website/esote.net /var/www/
rcctl reload httpd

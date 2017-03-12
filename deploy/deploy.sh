#!/bin/bash
# Copyright 2017 Eduardo Pinheiro (edpin@edpin.com). All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

BIN=upspin2tweet
ASSETS=assets
TEMPLATES=template

pushd ..
#GOOS=linux GOARCH=amd64 CGO_ENABLED=0
go build -ldflags="-s -w" || exit
upx $BIN


rsync -avz -e ssh ./$ASSETS deploy@upspin2tweet.com:/var/www/
rsync -avz -e ssh ./$TEMPLATES deploy@upspin2tweet.com:/var/www/
scp $BIN deploy@upspin2tweet.com:/tmp
ssh deploy@upspin2tweet.com "killall $BIN; cp /tmp/$BIN /var/www/$BIN; touch /tmp/errors; cd /var/www/; ./$BIN &>>/tmp/errors &"
popd
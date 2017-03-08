#!/bin/bash
# Copyright 2017 Eduardo Pinheiro (edpin@edpin.com). All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

BIN=storeserver
pushd $GOPATH/src/github.com/edpin/upspin2tweet/cmd/$BIN
go build -ldflags="-s -w" || exit
upx $BIN

scp $BIN deploy@upspin2tweet.com:/tmp
ssh deploy@upspin2tweet.com "killall $BIN; mv /tmp/$BIN /var/www/ephemeral/$BIN;"
ssh deploy@upspin2tweet.com "touch /tmp/ephemeral_errors; /var/www/ephemeral/$BIN -kind=inprocess -letscache=/etc/acme-cache/ -config=/var/www/ephemeral/config -https=:9999 &>>/tmp/ephemeral_errors &"
popd
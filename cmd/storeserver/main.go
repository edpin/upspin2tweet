// Copyright 2016 The Upspin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.upspin file.

package main

import (
	"net/http"

	"upspin.io/cloud/https"
	"upspin.io/config"
	"upspin.io/errors"
	"upspin.io/flags"
	"upspin.io/log"
	"upspin.io/rpc/storeserver"
	"upspin.io/serverutil/perm"
	"upspin.io/upspin"

	_ "upspin.io/transports"

	"upspin2tweet/lrustore"
)

const serverName = "ephemeral"

func main() {
	flags.Parse("addr", "config", "https", "kind", "letscache", "log", "project", "serverconfig", "tls")

	// Load configuration and keys for this server. It needs a real upspin username and keys.
	cfg, err := config.FromFile(flags.Config)
	if err != nil {
		log.Fatal(err)
	}

	// Create a new store implementation.
	var store upspin.StoreServer
	err = nil
	switch flags.ServerKind {
	case "inprocess":
		store, err = lrustore.New(flags.ServerConfig...)
	default:
		err = errors.Errorf("bad -kind %q", flags.ServerKind)
	}
	if err != nil {
		log.Fatalf("Setting up StoreServer: %v", err)
	}

	// Wrap with permission checks.
	ready := make(chan struct{})
	store, err = perm.WrapStore(cfg, ready, store)
	if err != nil {
		log.Fatalf("Error wrapping store: %s", err)
	}

	httpStore := storeserver.New(cfg, store, upspin.NetAddr(flags.NetAddr))
	http.Handle("/api/Store/", httpStore)
	https.ListenAndServeFromFlags(ready, serverName)
}

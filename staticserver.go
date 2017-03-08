// Copyright 2017 Eduardo Pinheiro (edpin@edpin.com). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// TODO(edpin): this was hacked from another project of mine. It's not very
// configurable; make it so (especially the caching stuff).
package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// NewStricFileServer creates a file server that does not allow listing
// directories nor cross-site linking. The SNI parameter identifies this
// site's domain, so that no one else can link to our assets. The file server
// also performs tagging and sets a cache control header for relevant assets.
func NewStricFileServer(root http.FileSystem, sni string) http.Handler {
	strictRoot := strictFileSystem{root}
	server := http.FileServer(strictRoot)
	return strictFileServer{
		Handler: server,
		root:    strictRoot,
		sni:     sni,
	}
}

type strictFileServer struct {
	http.Handler
	root http.FileSystem
	sni  string
}

// ServeHTTP implements net.Handler.
func (s strictFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ref := r.Referer()
	if strings.Contains(ref, "http://localhost") || strings.Contains(ref, s.sni) {
		s.setEtag(w, r)
		s.Handler.ServeHTTP(w, r)
	} else {
		log.Printf("Attempt to link to our assets from %q", ref)
		w.WriteHeader(http.StatusForbidden)
	}
}

// setEtag applies an Etag hash to all files in the file server. It also forces
// revalidation for JS and CSS files after 10 seconds and everything else after
// 5 minutes.
func (s strictFileServer) setEtag(w http.ResponseWriter, r *http.Request) {
	file, err := s.root.Open(r.URL.Path)
	if err != nil {
		log.Printf("Error opening %s: %s", r.URL.Path, err)
		return
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("Error reading all %s: %s", r.URL.Path, err)
		return
	}
	sum := sha256.Sum256(data)
	w.Header().Set("Etag", fmt.Sprintf("%x", string(sum[:])))
	if strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Cache-Control", "must-revalidate,max-age=10")
	} else {
		w.Header().Set("Cache-Control", "must-revalidate,max-age=300")
	}
}

type strictFileSystem struct {
	fs http.FileSystem
}

func (fs strictFileSystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	return noReaddirFile{f}, nil
}

type noReaddirFile struct {
	http.File
}

func (f noReaddirFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

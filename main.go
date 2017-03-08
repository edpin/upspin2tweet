// Copyright 2017 Eduardo Pinheiro (edpin@edpin.com). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	oauthTwitter "github.com/dghubble/oauth1/twitter"

	"upspin.io/access"
	"upspin.io/bind"
	"upspin.io/client"
	"upspin.io/cloud/https"
	"upspin.io/config"
	"upspin.io/errors"
	"upspin.io/log"
	"upspin.io/path"
	"upspin.io/rpc/dirserver"
	"upspin.io/rpc/storeserver"
	"upspin.io/serverutil/perm"
	"upspin.io/upspin"
	"upspin.io/valid"

	dirServer "upspin.io/dir/server"
	storeServer "upspin.io/store/server"

	// Load useful packers and transports.
	_ "upspin.io/pack/ee"
	_ "upspin.io/pack/eeintegrity"
	_ "upspin.io/pack/plain"
	_ "upspin.io/transports"
)

const (
	domainBase      = "upspin2tweet.com"
	serverUser      = "upspin@upspin2tweet.com"
	logDir          = "/var/www/dirserver-logs"
	upspinRoot      = "/var/www/upspinroot"
	upspinConfigDir = "/var/www/upspin"
	port            = ":8443" // must match file upspinConfigDir+"/config".
)

type server struct {
	config    oauth1.Config
	upspinCli upspin.Client
	keySrv    upspin.KeyServer
	storeSrv  upspin.StoreServer

	mu         sync.Mutex
	userSecret map[string]userGlue // map of twitterReqToken to userGlue
}

type userGlue struct {
	UpspinUser upspin.UserName
	Token      *oauth1.Token // the access token.

	secret string // twitter user secret (oauth1)
}

var (
	testing      = flag.Bool("testing", false, "whether running on localhost only")
	upspinCfgDir = flag.String("config", upspinConfigDir, "path to upspin config file")
)

func main() {
	flag.Parse()

	if *testing {
		*upspinCfgDir = filepath.Join(os.Getenv("HOME"), "upspin")
	}

	upspinCfg, err := config.FromFile(filepath.Join(*upspinCfgDir, "config"))
	if err != nil {
		panic(err)
	}

	s := &server{
		config: oauth1.Config{
			ConsumerKey:    twitterConsumerKey,
			ConsumerSecret: twitterConsumerSecret,
			CallbackURL:    "https://upspin2tweet.com/twitterauth",
			Endpoint:       oauthTwitter.AuthorizeEndpoint,
		},
		userSecret: make(map[string]userGlue),
		upspinCli:  client.New(upspinCfg),
	}
	s.keySrv, err = bind.KeyServer(upspinCfg, upspinCfg.KeyEndpoint())
	if err != nil {
		panic(err)
	}

	// Set up HTTPS server.
	opt := &https.Options{
		LetsEncryptCache: "/etc/acme-cache/",
		LetsEncryptHosts: []string{domainBase, "upspin.upspin2tweet.com"},
	}

	http.HandleFunc("/authorize", s.handleLogin)
	http.HandleFunc("/twitterauth", s.handleCallback)
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/welcome", s.handleWelcome)
	http.Handle("/assets/", NewStricFileServer(http.Dir("."), domainBase))
	http.Handle("/favicon.ico", http.RedirectHandler("/assets/favicon.png", http.StatusMovedPermanently))

	if *testing {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}

	ready := make(chan struct{})
	err = s.startUpspinServer(ready)
	if err != nil {
		panic(err)
	}
	go s.watchAndTweet()

	log.Printf("Starting up...")
	https.ListenAndServe(ready, "", port, opt)
}

func (s *server) startUpspinServer(ready chan struct{}) error {
	cfg, err := config.FromFile(filepath.Join(*upspinCfgDir, "config"))
	if err != nil {
		return err
	}

	// Set up StoreServer.
	store, err := storeServer.New("backend=Disk", "basePath="+upspinRoot)
	if err != nil {
		return err
	}
	store, err = perm.WrapStore(cfg, ready, store)
	if err != nil {
		return fmt.Errorf("error wrapping store: %s", err)
	}
	s.storeSrv = store

	// Set up DirServer.
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return err
	}
	dir, err := dirServer.New(cfg, "userCacheSize=1000", "logDir="+logDir)
	if err != nil {
		return err
	}
	dir, err = perm.WrapDir(cfg, ready, serverUser, dir)
	if err != nil {
		return err
	}

	// Set up RPC server.
	httpStore := storeserver.New(cfg, store, cfg.StoreEndpoint().NetAddr)
	httpDir := dirserver.New(cfg, dir, cfg.DirEndpoint().NetAddr)
	http.Handle("/api/Store/", httpStore)
	http.Handle("/api/Dir/", httpDir)
	return nil
}

func (s *server) handleHome(w http.ResponseWriter, r *http.Request) {
	headerTpl.Execute(w, "")
	homeTpl.Execute(w, "")
	footerTpl.Execute(w, "")
}

var headerTpl = template.Must(template.New("header").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
<link rel="stylesheet" href="/assets/css/bootstrap.min.css">
<link rel="stylesheet" type="text/css" href="https://fonts.googleapis.com/css?family=Droid+Sans+Mono">
</head>
<body>
`))

var footerTpl = template.Must(template.New("footer").Parse(`
</body>
</html>
`))

var homeTpl = template.Must(template.New("home").Parse(`
<div class="container">
<div class="row">
<div class="col-md-2"></div>
<div class="col-md-8">
	<h1>Upspin 2 Tweet</h1>
	<h3>Tweet by simply writing to an <a href="https://upspin.io">Upspin</a> file</h3>
</div>
<div class="col-md-2"></div>
</div>

<div class="row">
<div class="col-md-2"></div>
<div class="col-md-8">
	<img src="/assets/augie-tweet.png">
</div>
<div class="col-md-2"></div>
</div>

<br>
<div class="row">
<div class="col-md-2"></div>
<div class="col-md-6">
	<p>Sign up by associating your Upspin user name with your Twitter account.</p>
	<form action="/authorize" method="get">
    	<div class="form-group">
       		<label for="emailId">Upspin User Name</label>
       		<input type="email" class="form-control" witdth="40" id="emailId" name="upspinusername" placeholder="ann@example.com">
    	</div>
    	<div class="col-md-4"></div>
    	<div class="col-md-2"><input type="image" src="/assets/sign-in-with-twitter-gray.png"></div>
    	</form>
</div>
<div class="col-md-4"></div>
</div>
<br>
<br>
<div class="row">
<div class="col-md-2"></div>
<div class="col-md-8">
	<h3>Frequently Asked Questions</h3>
	<p></p>
	<p><b>Q: How does it work?</b></p>
	<p>A: You associate an Upspin user name to your Twitter account. Then
	you can tweet by simply writing to a special Upspin file that only you
	control. Here's an example:
	<pre>echo "I'm tweeting" | upspin put upspin@upspin2tweet.com/[twitter name]/tweet</pre>
	<p><b>Q: Do you need my Twitter password?</b></p>
	<p>A: No. We use OAuth and simply store a revokable auth token.</p>
	<p><b>Q: How do you store my Twitter auth token?</b></p>
	<p>A: We store it in an encrypted Upspin file served by this server.
	   Only you and the server can see your token.</p>
	<p><b>Q: Do I need to have my own Upspin servers running?</b></p>
	<p>A: No. You only need a place to store your tweet temporarily while we
	post it to Twitter. If you don't have a StoreServer associated with your
	Upspin username, you can use ours. Just add this to your upspin/config
	file:
	<pre>storeserver: remote,ephemeral.upspin2tweet.com</pre>
	<p><b>Q: How do I delete my account?</b><p>
	<p>A: Delete your Token and your account will be removed. Do this:</p>
	<pre>upspin rm upspin2tweet.com/[your twitter name]/cfg/Token</pre>
	<p><b>Q: Can I share or delegate my twitter account with other Upspin accounts?</b></p>
	<p>A: Yes. Just sign in with Twitter again using the button above and
	   use a different Upspin user name and the new user name will be added
	   to the Access file of the authorized Twitter account.</p>
	<p><b>Q: Can I remove an Upspin user from my Twitter account?</b></p>
	<p>A: No. But you can delete your account completely (see
	   above) and create it again from stracth and only add those Upspin
	   users you want to have access to your Twitter.</p>
</div>
<div class="col-md-2"></div>
</div>
<br><br>
<div class="row">
<div class="col-md-2"></div>
<div class="col-md-8">
	<h6>
	Upspin mascot <a href="https://upspin.io/doc/mascot.md">Augie</a> is
	copyright by <a href="https://www.instagram.com/reneefrench/">Renne
	French</a>. Twitter logo copyright of Twitter Inc. Web UI built using
	<a href="http://getbootstrap.com">Bootstrap</a>. All Rights Reserved.
	</h6>
</div>
<div class="col-md-2"></div>
</div>
`))

var welcomeTpl = template.Must(template.New("welcome").Parse(`
<style>
pre {
    display: inline-block;
}
</style>

<div class="container">

<div class="row">
<h1>Welcome to upspin2tweet, <em>{{.TwitterName}}</em></h1>
</div>

<br>

<div class="row">
<p>To begin tweeting, simply write your tweet to this upspin file:</p>
<pre>{{.ServerName}}/{{.TwitterName}}/tweet</pre>
<p>The server will tweet on your behalf and will erase the contents of the file
named above.</p>
<p>If for any reason you're no longer comfortable with us having your twitter
auth token, you can delete it by removing the following file:<p>
<div><pre>{{.ServerName}}/{{.TwitterName}}/cfg/Token</pre></div>
<p>You can also cancel this app's authority to use your auth token, by visiting:<p>
<p><a href="https://twitter.com/settings/applications">https://twitter.com/settings/applications</a><p>
<p>If you don't have a store server to use, you can use our ephemeral one. Simply add this to your config file:</p>
<pre>storeserver: remote,ephemeral.upspin2tweet.com</pre>
<br>
<h3>Happy tweeting and upspinning!</h3>
</div>

</div>
`))

func (s *server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	twitterName := r.URL.Query().Get("twittername")
	data := struct {
		ServerName  string
		TwitterName string
	}{
		ServerName:  serverUser,
		TwitterName: twitterName,
	}
	headerTpl.Execute(w, "")
	welcomeTpl.Execute(w, data)
	footerTpl.Execute(w, "")
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("upspinusername")
	if name == "" {
		fmt.Fprint(w, "Please supply your upspin user name")
		return
	}
	userName := upspin.UserName(name)
	err := valid.UserName(userName)
	if err != nil {
		fmt.Fprintf(w, "Invalid upspin user name: %s", userName)
		return
	}
	// Manually exclude access.All until issue #262 is fixed.
	if userName == access.AllUsers {
		fmt.Fprintf(w, "Invalid upspin user name all", userName)
		return
	}
	_, err = s.keySrv.Lookup(userName)
	if err != nil && errors.Match(errors.E(errors.NotExist), err) {
		fmt.Fprintf(w, "Unknown upspin user name: %s", userName)
		return
	}

	reqToken, requestSecret, err := s.config.RequestToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	s.mu.Lock()
	s.userSecret[reqToken] = userGlue{
		secret:     requestSecret,
		UpspinUser: userName,
	}
	s.mu.Unlock()

	authorizationURL, err := s.config.AuthorizationURL(reqToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, authorizationURL.String(), http.StatusFound)
}

func (s *server) handleCallback(w http.ResponseWriter, r *http.Request) {
	checkErr := func(prefix string, err error) error {
		if err == nil {
			return nil
		}
		var errMsg string
		if prefix != "" {
			errMsg = prefix
		}
		errMsg += err.Error()
		http.Error(w, errMsg, http.StatusInternalServerError)
		log.Error.Print(errMsg)
		return err
	}

	reqToken, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if checkErr("twitterCallback:", err) != nil {
		return
	}
	s.mu.Lock()
	glue, ok := s.userSecret[reqToken]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "twitter token not found", http.StatusInternalServerError)
		return
	}
	reqSecret := glue.secret

	accessToken, accessSecret, err := s.config.AccessToken(reqToken, reqSecret, verifier)
	if checkErr("Error in callback", err) != nil {
		return
	}

	// Configure an HTTP client for this Twitter user.
	glue.Token = oauth1.NewToken(accessToken, accessSecret)
	screenName, err := s.configureTwitterUser(glue)
	if checkErr("configureTwitterUser:", err) != nil {
		return
	}
	http.Redirect(w, r, "/welcome?twittername="+screenName, http.StatusFound)
}

// addUserToAccessFile adds a user to the named Access file if it's not yet
// there. If the Access file does not exist, it is created. The server user is
// always added when the file is created.
func (s *server) addUserToAccessFile(accessFile upspin.PathName, user upspin.UserName, rights string) error {
	data, err := s.upspinCli.Get(accessFile)
	if err != nil {
		if !errors.Match(errors.E(errors.NotExist), err) {
			return err
		}
		// New file. Create it.
		_, err = s.upspinCli.Put(accessFile, []byte(fmt.Sprintf("*:%s\n%s:%s", serverUser, rights, user)))
		return err
	}
	acc, err := access.Parse(accessFile, data)
	if err != nil {
		return err
	}
	users := acc.List(access.AnyRight)
	for _, u := range users {
		if u.User() == user {
			// Already there, nothing else to do.
			return nil
		}
	}
	// No there yet. Add it now.
	_, err = s.upspinCli.Put(accessFile, []byte(fmt.Sprintf("%s\n%s:%s", data, rights, user)))
	return err
}

// configureTwitterUser return the user's screen name associated with the token
// and creates a directory for them.
func (s *server) configureTwitterUser(glue userGlue) (string, error) {
	httpClient := s.config.Client(oauth1.NoContext, glue.Token)
	client := twitter.NewClient(httpClient)
	param := twitter.AccountVerifyParams{}
	user, _, err := client.Accounts.VerifyCredentials(&param)
	if err != nil {
		return "", err
	}

	// Create a subdir under the user's screen name as follows:
	// /Access      -- Allows users to list, read and write, but not create.
	// /tweet       -- File the user will write to to tweet.
	// /cfg/Access  -- Only the server can see.
	// /cfg/Token   -- The twitter OAuth token
	newUserDir := upspin.PathName(serverUser + "/" + user.ScreenName)
	_, err = s.upspinCli.MakeDirectory(newUserDir)
	if err != nil && !errors.Match(errors.E(errors.Exist), err) {
		return "", err
	}
	_, err = s.upspinCli.MakeDirectory(path.Join(newUserDir, "cfg"))
	if err != nil && !errors.Match(errors.E(errors.Exist), err) {
		return "", err
	}
	// Add user glue.UpspinUser with rights "r,l,d".
	err = s.addUserToAccessFile(path.Join(newUserDir, "cfg", "Access"), glue.UpspinUser, "r,l,d")
	if err != nil {
		return "", err
	}
	tokenBlob, err := json.Marshal(glue)
	if err != nil {
		return "", err
	}
	_, err = s.upspinCli.Put(path.Join(newUserDir, "cfg", "Token"), []byte(tokenBlob))
	if err != nil {
		return "", err
	}
	_, err = s.upspinCli.Put(path.Join(newUserDir, "tweet"), []byte(""))
	if err != nil {
		return "", err
	}
	// Add user glue.UpspinUser with rights "r,l,w".
	err = s.addUserToAccessFile(path.Join(newUserDir, "Access"), glue.UpspinUser, "r,l,w")
	if err != nil {
		return "", err
	}
	return user.ScreenName, nil
}

func (s *server) watchAndTweet() {
	serverRoot := upspin.PathName(serverUser + "/")
	dir, err := s.upspinCli.DirServer(serverRoot)
	if err != nil {
		panic(err)
	}

	for {
		done := make(chan struct{})
		events, err := dir.Watch(serverRoot, -1, done)
		if err != nil {
			log.Error.Printf("Can't Watch dir root %s: %s", serverUser, err)
			close(done)
			time.Sleep(1 * time.Second)
			continue
		}
		s.watch(events, done)
	}
}

func (s *server) watch(events <-chan upspin.Event, done chan struct{}) {
	defer close(done)
	for {
		e, ok := <-events
		if !ok {
			log.Printf("Channel closed.")
			return
		}
		if e.Error != nil {
			log.Error.Printf("Error event: %s", e.Error)
			return
		}
		if e.Entry == nil {
			log.Error.Printf("Got nil Entry (%+v) Server is crazy", e)
			return
		}
		if e.Delete {
			log.Printf("Got a delete: %s", e.Entry.Name)
			s.maybeRemoveUserAccount(e.Entry)
			continue
		}
		log.Printf("Got event: %s", e.Entry.Name)
		if !strings.HasSuffix(string(e.Entry.Name), "/tweet") {
			// Not a tweet. Ignore.
			continue
		}
		go s.tweet(e.Entry) // Logs on error.
	}
}

func (s *server) tweet(entry *upspin.DirEntry) {
	var tweet []byte
	var err error
	// Try to read the tweet. Due to the Upspin cache, backoff if we can't
	// read it and try again.
	for i := 1; i <= 5; i++ {
		tweet, err = s.upspinCli.Get(entry.Name)
		if err == nil {
			break
		}
		log.Error.Printf("Could not read tweet: %s", err)
		time.Sleep(100 * time.Duration(i*i) * time.Millisecond)
	}
	if err != nil {
		// Failed to read.
		return
	}
	if len(tweet) == 0 {
		// Nothing to do.
		log.Debug.Printf("Nothing to do. Empty tweet in %s", entry.Name)
		return
	}
	// Since we have a tweet, we should remove it once we process it, even
	// if errors occur.
	defer s.cleanup(entry)

	// Read the tweeterer's config.
	p, _ := path.Parse(entry.Name)
	dir := p.Drop(1)
	cfg, err := s.upspinCli.Get(path.Join(dir.Path(), "cfg", "Token"))
	if err != nil {
		log.Error.Printf("Could not read config: %s", err)
		return
	}
	var userCfg userGlue
	err = json.Unmarshal(cfg, &userCfg)
	if err != nil {
		log.Error.Printf("json.Unmarshal: %q: %s", cfg, err)
		return
	}

	// Finally, tweet it out.
	httpClient := s.config.Client(oauth1.NoContext, userCfg.Token)
	client := twitter.NewClient(httpClient)
	var status twitter.StatusUpdateParams
	_, _, err = client.Statuses.Update(string(tweet), &status)
	if err != nil {
		log.Error.Printf("Error tweeting: %s", err)
		return
	}
	log.Printf("Server successfully tweeted.")
}

func (s *server) cleanup(entry *upspin.DirEntry) {
	// Don't delete the file or the user loses access. Instead, put a
	// zero-length file to mark there's nothing else to do.
	_, err := s.upspinCli.Put(entry.Name, []byte(""))
	if err != nil {
		log.Error.Printf("Can't delete %s: %s", entry.Name, err)
	}
	// Remove the underlying blocks too, if they're hosted on our ephemeral
	// store server.
	for _, b := range entry.Blocks {
		if !strings.Contains(string(b.Location.Endpoint.NetAddr), domainBase) {
			continue
		}
		err = s.storeSrv.Delete(b.Location.Reference)
		if err != nil {
			log.Error.Printf("Can't delete ref: %s", b.Location.Reference)
		}
	}
}

// maybeRemoveUserAccount removes the subdirectory for the twitter account
// associated with the entry that was removed, if the entry is for a /Token
// file.
func (s *server) maybeRemoveUserAccount(entry *upspin.DirEntry) {
	if !strings.HasSuffix(string(entry.Name), "/cfg/Token") {
		// Likely the super user doing some maintenance.
		log.Debug.Printf("Not a Token file.")
		return
	}
	p, err := path.Parse(entry.Name)
	if err != nil {
		log.Error.Printf("maybeRemoveUserAccount: %s", err)
		return
	}
	p = p.Drop(2)
	log.Printf("Removing everything under %s", p.Path())
	s.removeAll(p.Path(), false)
}

// TODO(edpin): this was an attempt of a more generic RemoveAll function. It's
// way overkill for upspin2tweet since there are no links. Simplify.
func (s *server) removeAll(name upspin.PathName, followLink bool) error {
	const op = "RemoveAll"

	e, err := s.upspinCli.Lookup(name, followLink)
	if err == upspin.ErrFollowLink {
		// Only happens when followLink is false. In this case, we must
		// remove the link itself and be done.
		err = s.upspinCli.Delete(e.Link)
		if err != nil {
			return errors.E(op, err)
		}
		return nil
	}
	if err != nil {
		return errors.E(op, err)
	}
	if !e.IsDir() {
		log.Printf("Deleting %s", e.Name)
		err = s.upspinCli.Delete(e.Name)
		if err != nil {
			return errors.E(op, err)
		}
		return nil
	}

	entries, err := s.upspinCli.Glob(string(e.Name) + "/*")
	if err == upspin.ErrFollowLink {
		panic("wtf?")
	}
	if err != nil {
		return errors.E(op, err)
	}
	for _, ent := range entries {
		err = s.removeAll(ent.Name, false) // do not follow links.
		if err != nil {
			return err // No need to re-wrap.
		}
	}
	// Remove the top directory as well.
	err = s.upspinCli.Delete(e.Name)
	if err != nil {
		return errors.E(op, err)
	}
	return nil
}
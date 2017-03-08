# upspin2tweet

Connects Upspin users to a Twitter account. See live app at
[upspin2tweet.com](https://upspin2tweet.com).

This is work in progress and the code was put together over a weekend. Lots
remain to clean it up and make it more presentable. Feel free to send me pull
requests to clean up. In fact, I'd appreciate if you sent fixes for errors and
omissions you may find, especially if they pertain to security.

This is the full code backing up the live app, except for:
- a single config file that contains the server's Twitter auth secrets.
- the server's Upspin keys and config files.
- the reverse proxy configuration file (so we can run everything on port 443).


Things TODO:
- Pull out the HTML templates from the code.
- Change a bunch of hard-coded paths in `main.go` to flags.
- Better deploy scripts (perhaps, Dockerize it?).

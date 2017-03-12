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

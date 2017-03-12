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
	<h5><a href="/privacy">Privacy Policy</a></h5>
	<h6>
	Upspin mascot <a href="https://upspin.io/doc/mascot.md">Augie</a> is
	copyright by <a href="https://www.instagram.com/reneefrench/">Renne
	French</a>. Twitter logo copyright of Twitter Inc. Web UI built using
	<a href="http://getbootstrap.com">Bootstrap</a>. All Rights Reserved.
	</h6>
</div>
<div class="col-md-2"></div>
</div>

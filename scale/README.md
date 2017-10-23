### scale testing notes

Refer to the [Locust](https://locust.io/) for more info on Locust, and
consult the documentation on how to install it.

For playing around, ran the provided sample by first 
starting the sample app, then locust:

<pre>
go run main.go --port 8080
</pre>

<pre>
locust -f simple.py --host http://localhost:8080
</pre>

I then opened my browser on the local url (printed to standard out, for me it 
was localhost:8089), and dialed up the transaction to the 18.6 - 19.8 RPS
rate, which was 65 user. This will run at a steady rate with no errors.

I then pushed it to 75 users, which resulted in HTTP 429 errors.

I was also able to do the same thing running two servers and running 2 locust sessions, observing
the same behavior for the same totals of users.

<pre>
go run main --port 8080&
go run main --port 9090&
locust -f simple.py --host http://localhost:8080&
locust -f simple.py --host http://localhost:9090 -P 8091&
</pre>
# rrl - rolling rate limiter

Golang adaption of the ClassDojo Engineering [rolling rate limiter](https://engineering.classdojo.com/blog/2015/02/06/rolling-rate-limiter/)

This project implements a distributed, rolling rate limiter that that
uses a Redis transaction to group the set of operations needed to
determine if the current operation is allowed based on the request
budget and recent activity.

This implementation implements the algorithm in a Lua script that is
executed on the Redis cluster. The implementation does not count 
disallowed requests against the request budget, so as load goes above
the allowed threshold only that portion of the calls above the threshold
are disallowed.

A sample use of the rate limiter is supplied in the sample directory, and 
some materials related to scale testing the implementation using locust
are supplied in the scale directory.

## Contributing

To contribute, you must certify you agree with the [Developer Certificate of Origin](http://developercertificate.org/)
by signing your commits via `git -s`. To create a signature, configure your user name and email address in git.
Sign with your real name, do not use pseudonyms or submit anonymous commits.

## License

(c) 2017 Fidelity Investments
Licensed under the Apache License, Version 2.0
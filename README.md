# about

This is an interactive test harness and runner for test262.

# expected use

Clone test262 into a directory 'test262' inside this repository, and 'go run
main.go'

# future work

* Run older test262 too (for ES5 compatibility checking)
* Command line runner?
* Cleanup (I know it's messy right now)
* Testing (I know it's fragile right now)
* Save information on the last test run, for comparison against a new run
* Better statistics reporting
* Comments in TestExpectations
* Expected failures (similar to v4's old runner)

# Ratelimited

Just a simple wrapper over [ratelimit](https://github.com/uber-go/ratelimit). :)

## Example

See a working example [here](github.com/pythonista7/ratelimited/example/example.go). But here's the idea:

```go
func main() {
	const targetRPM int = 100
    queue := make(chan (string), 25)
    /*
    This will create an instance of the ratelimitedworker.
    - id: primary identifier for the ratelimiter Job the work is assosiated to
    - targetRPM: the expected maximum rate limit to perform at
    - hasty: if true maintains a higher RPS good for work() that takes long, false forces rate below the limit.
    - verbose: enable logging
    */
    rlc := ratelimitedworker.Create("sampleId", targetRPM, true, true)
    go queueLoader(queue)
    doLotsOfWork(rlc,queue)
}

func doLotsOfWork(rlc *ratelimitedworker.RLCtr, queue chan (string)) {

	for range queue {
		// this will allow the work() to be called only rlc.targetRPM number of times during a minute
		rlc.Track() // if limit is hit for the time period it will block the below go routine
		go work()
	}
}

```
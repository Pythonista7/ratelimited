package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	ratelimitedworker "github.com/pythonista7/ratelimited"
)

const dateFormatTemplate = "01-02-2006 15:04:05"

var delayList = [...]int{100, 200, 500, 1000}

func main() {
	var wg sync.WaitGroup

	queue := make(chan (string), 25)

	wg.Add(2)
	go producer(&wg, queue)
	go consumer(&wg, queue)
	wg.Wait()
}

func producer(wg *sync.WaitGroup, queue chan (string)) {
	ticker := time.NewTicker(50 * time.Millisecond)
	done := make(chan bool)
	fmt.Println("Started producer")
	go func() {
		for {
			select {
			case <-done:
				wg.Done()
				return
			case t := <-ticker.C:
				queue <- "Custom Payload: " + t.Format(dateFormatTemplate)
				if len(queue) > 0 && len(queue)%10 == 0 {
					fmt.Printf("Buffer = %d/%d\n", len(queue), cap(queue))
				}
			}
		}
	}()
}

func consumer(wg *sync.WaitGroup, queue chan (string)) {
	const targetRPM int = 1000

	rlw := ratelimitedworker.Create("sampleId", targetRPM, true, true)

	go func() {
		defer wg.Done()
		makeReqs(&rlw, queue)
	}()
}

func makeReqs(rlw *ratelimitedworker.RLW, queue chan (string)) {

	for range queue {
		// this will allow the work() to be called rlw.targetRPM number of times only
		rlw.Track() // if limit is hit for the time period it will block this go routine
		go work()
	}
}

func work() {
	// do any network call / other job here
	delay := delayList[rand.Intn(len(delayList))]
	time.Sleep(time.Duration(delay) * time.Millisecond)

}

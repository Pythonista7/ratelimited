package ratelimitedworker

import (
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"go.uber.org/ratelimit"
)

type RLCtr struct {
	ID        string
	RL        ratelimit.Limiter
	TargetRPM uint32

	history   uint32
	verbose   bool        // enables logging
	logger    *log.Logger // pkg logger instance
	targetRPS int         // internally computed from TargetRPM and `hasty` param in Create()
	active    bool        // bool to flag if the rlc has started
	// TODO: maybe have a channel which can send the message to stop proccessing
	// TODO: work on cleanup
}

/*
This will create an instance of the ratelimitedworker.

- id: primary identifier for the ratelimiter Job the work is assosiated to

- targetRPM: the expected maximum rate limit to perform at

- hasty: if true maintains a higher RPS good for work() that takes long, false forces rate below the limit.

- verbose: enable logging
*/
func Create(
	id string,
	targetRPM int,
	hasty bool,
	verbose bool,
) RLCtr {

	var targetRPS int
	var logger *log.Logger

	if hasty {
		targetRPS = int(math.Ceil(float64(targetRPM / 60)))
	} else {
		targetRPS = int(math.Floor(float64(targetRPM / 60)))
	}

	if verbose {
		logger = log.Default()
		logPrefix := fmt.Sprintf("RatelimitedWorkerId : %s | ", id)
		logger.SetPrefix(logPrefix)
		logger.SetFlags(log.Ltime | log.Lshortfile)
	} else {
		logger = nil
	}

	return RLCtr{
		ID:        id,
		RL:        ratelimit.New(targetRPS), // param is rps
		TargetRPM: uint32(targetRPM),
		verbose:   verbose,
		history:   0,
		logger:    logger,
		targetRPS: targetRPS,
	}
}

func (rlc *RLCtr) Track() {
	if !rlc.active {
		// init
		rlc.start()
		if rlc.verbose {
			rlc.logger.Println("Init Success")
		}
	}

	// in case we hit the per min rate limit too soon , we wait around until we can resume
	for rlc.history >= rlc.TargetRPM {
		if rlc.verbose {
			rlc.logger.Printf(
				"Reached RPM waiting for history refresh: {history : %d , allowedRPM: %d }\n",
				rlc.history, rlc.TargetRPM,
			)
		}

		time.Sleep(1 * time.Second)
	}

	rlc.RL.Take()
	atomic.AddUint32(&rlc.history, 1)

}

func (rlc *RLCtr) start() {
	rlc.active = true
	window := 5

	// optional logger ticker just for understanding
	go func() {
		ticker := time.NewTicker(time.Duration(window) * time.Second)
		prev := rlc.history
		for range ticker.C {

			var counter uint32
			if rlc.history > prev {
				counter = rlc.history - prev
			} else {
				// when Hasty and rate limit is hit for the minute is hit before 60s ,
				// we need to reset this because we're using uint
				counter = 0
			}

			if rlc.verbose {
				rlc.logger.Printf(
					"Completed %d tasks in the last window(%d seconds) , targetRPS = %d , currentRPS = %d , completedTaskCountThisMinute = %d \n",
					counter, window, rlc.targetRPS, counter/uint32(window), rlc.history)
			}

			// update prev to track window counter
			prev = rlc.history

		}
	}()

	// essential, this resets history and helps stay within RPMLimit
	go func() {
		monitor := time.NewTicker(1 * time.Minute)
		for range monitor.C {
			if rlc.verbose {
				rlc.logger.Printf(
					"Period Stats(last 1 minute): noOfReqSent = %d , rateLimit(per min) = %d \n",
					rlc.history, rlc.TargetRPM,
				)
			}

			// reset history every minute
			atomic.StoreUint32(&rlc.history, 0)

		}
	}()
}

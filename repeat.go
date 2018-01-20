package uhttp

import (
	"math/rand"
	"time"
)

// RepeatFunc generates delays between successive repeats of a request.  prev will
// contain the value of the previous repeat (zero for the first).  It should return
// nil to stop repeating, and should not be called after that.
type RepeatFunc func(prev time.Duration) *time.Duration

// RepeatGenerator produces RepeatFuncs.
type RepeatGenerator func() RepeatFunc

// RepeatAfter generates a RepeatFunc that returns delay num times, and then returns nil.
// If num is <= 0, this will return delay forever.
func RepeatAfter(delay time.Duration, num int) RepeatGenerator {
	if num == 0 {
		num = -1
	}
	return func() RepeatFunc {
		num := num
		return func(_ time.Duration) *time.Duration {
			switch {
			case num > 0:
				num--
				fallthrough
			case num < 0:
				return &delay
			default:
				return nil
			}
		}
	}
}

// RepeatJoin generates a RepeatFunc that iterates over gens, instantiates RepeatFuncs from them,
// and returns values from each until they return nil, at which point it moves on to the next
// RepeatFunc.  Returns nil once all resulting RepeatFunc instances have returned nil.
func RepeatJoin(gens ...RepeatGenerator) RepeatGenerator {
	return func() RepeatFunc {
		var i int
		var fn RepeatFunc
		return func(prev time.Duration) *time.Duration {
			var got *time.Duration
			for i < len(gens) {
				if fn == nil { // "start" the repeater
					fn = gens[i]()
				}
				if got = fn(prev); got != nil {
					return got
				}
				i++
				fn = nil
			}
			return nil
		}
	}
}

// randomDuration returns a duration between low and high.
func randomDuration(low, high time.Duration) time.Duration {
	if low > high {
		low, high = high, low
	}
	return time.Duration(rand.Int63n(int64(high)-int64(low)) + int64(low))
}

// RepeatRandom generates a RepeatFunc that returns a random duration between [low, high), num
// times, and then returns nil.  If num is 0, this will return delays forever.
func RepeatRandom(low, high time.Duration, num int) RepeatGenerator {
	if num == 0 {
		num = -1
	}
	return func() RepeatFunc {
		num := num
		return func(_ time.Duration) *time.Duration {
			switch {
			case num > 0:
				num--
				fallthrough
			case num < 0:
				d := randomDuration(low, high)
				return &d
			default:
				return nil
			}
		}
	}
}

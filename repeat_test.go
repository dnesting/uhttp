package uhttp

import (
	"fmt"
	"testing"
	"time"
)

func TestRepeat(t *testing.T) {
	a1 := RepeatAfter(time.Duration(1), 1)
	a2 := RepeatAfter(time.Duration(2), 2)
	a3 := RepeatRandom(time.Duration(5), time.Duration(10), 2)
	joined := RepeatJoin(a1, a2, a3)
	// expected: [1 2 2 5-10]

	// Repeat this test twice to be sure that the closures work as expected.
	doTestRepeat(t, "first", joined())
	doTestRepeat(t, "second", joined())
}

func doTestRepeat(t *testing.T, desc string, fn RepeatFunc) {
	var actual []int64
	for count := 0; count < 10; count++ {
		if r := fn(time.Duration(0)); r != nil {
			actual = append(actual, int64(*r))
		} else {
			break
		}
	}

	if len(actual) != 5 || actual[0] != 1 || actual[1] != 2 || actual[2] != 2 || (actual[3] < 5 || actual[3] >= 10) || (actual[4] < 5 || actual[4] >= 10) {
		t.Errorf("(%s) expected [1 2 2 5-10 5-10], got %v", desc, actual)
	}
}

func TestRepeatAfterever(t *testing.T) {
	gen := RepeatAfter(time.Duration(1), 0)
	doTestRepeatAfterever(t, "first", "[1 1 1 1 1]", gen())
	doTestRepeatAfterever(t, "second", "[1 1 1 1 1]", gen())
}

func doTestRepeatAfterever(t *testing.T, desc string, expected string, fn RepeatFunc) {
	var act []int64
	for count := 0; count < 5; count++ {
		if r := fn(time.Duration(0)); r != nil {
			act = append(act, int64(*r))
		} else {
			break
		}
	}
	actual := fmt.Sprintf("%v", act)
	if expected != actual {
		t.Errorf("(%s) expected %q, got %q", desc, expected, actual)
	}
}

func ExampleRepeatJoin() {
	tenHzFor3 := RepeatAfter(time.Second/10, 3)
	oneHz := RepeatAfter(time.Second, 0)

	joined := RepeatJoin(tenHzFor3, oneHz)()

	for i := 0; i < 6; i++ {
		next := joined(0)
		if next == nil {
			break
		}
		fmt.Println(*next)
	}

	// Output:
	// 100ms
	// 100ms
	// 100ms
	// 1s
	// 1s
	// 1s
}

package teian

import "time"

type Timer struct {
	*time.Timer
	hour   int
	minute int
	second int
}

func nextTick(hour, minute, second int) time.Duration {
	now := time.Now()
	nextTick := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, time.Local)
	if nextTick.Before(now) {
		nextTick = nextTick.Add(24 * time.Hour)
	}
	return nextTick.Sub(time.Now())
}

func NewTimer(hour, minute, second int) Timer {
	return Timer{
		time.NewTimer(nextTick(hour, minute, second)),
		hour,
		minute,
		second,
	}
}

func (t Timer) Reset() {
	t.Timer.Reset(nextTick(t.hour, t.minute, t.second))
}

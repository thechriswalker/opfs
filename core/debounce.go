package core

import (
	"time"
)

//taken and modified from underscore.go
//but I only need the one function
func Debounce(fn func(), wait time.Duration) func() {
	var timestamp time.Time
	var later func()
	var timer *time.Timer = nil
	later = func() {
		last := time.Now().Sub(timestamp)
		if last < wait && last > 0 {
			go func() {
				timer = time.NewTimer(wait - last)
				for {
					select {
					case <-timer.C:
						timer.Stop()
						later()
					}
				}
			}()
		} else {
			fn()
			timer.Stop()
		}
	}

	return func() {
		timestamp = time.Now()
		if timer == nil {
			timer = time.NewTimer(wait)
			go func() {
				for {
					<-timer.C
					timer.Stop()
					later()
					break
				}
			}()
		}
	}
}

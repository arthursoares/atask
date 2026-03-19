package service

import "time"

func timeNow() time.Time {
	return time.Now().UTC()
}

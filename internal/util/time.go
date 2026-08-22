package util

import "time"

func Unix(t time.Time) int64 {
	return t.UTC().Unix()
}

func Timestamp(value int64) time.Time {
	return time.Unix(value, 0).UTC()
}

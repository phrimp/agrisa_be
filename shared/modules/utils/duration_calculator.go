package utils

import "time"

func durationUntilNextMidnight() time.Duration {
	now := time.Now()

	year, month, day := now.Date()

	todayMidnight := time.Date(year, month, day, 0, 0, 0, 0, now.Location())

	nextMidnight := todayMidnight.Add(24 * time.Hour)

	return nextMidnight.Sub(now)
}

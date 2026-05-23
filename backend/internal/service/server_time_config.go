package service

import (
	"os"
	"strconv"
	"time"
)

type ServerTimeConfig struct {
	Location     *time.Location
	RolloverHour int
}

func LoadServerTimeConfig() ServerTimeConfig {
	loc := time.UTC
	if tz := os.Getenv("ANTRADER_SERVER_TIMEZONE"); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}

	hour := 0
	if v := os.Getenv("ANTRADER_ROLLOVER_HOUR"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < 0 {
				n = 0
			}
			if n > 23 {
				n = 23
			}
			hour = n
		}
	}

	return ServerTimeConfig{Location: loc, RolloverHour: hour}
}

func NextRolloverUTC(now time.Time, cfg ServerTimeConfig) time.Time {
	loc := cfg.Location
	if loc == nil {
		loc = time.UTC
	}

	localNow := now.In(loc)
	y, m, d := localNow.Date()
	candidate := time.Date(y, m, d, cfg.RolloverHour, 0, 0, 0, loc)
	if !candidate.After(localNow) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate.UTC()
}

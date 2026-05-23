package service

import (
	"os"
	"sort"
	"strings"
	"time"
)

func timeframeDuration(tf string) time.Duration {
	switch strings.ToLower(tf) {
	case "m1", "1m":
		return time.Minute
	case "m5", "5m":
		return 5 * time.Minute
	case "m15", "15m":
		return 15 * time.Minute
	case "m30", "30m":
		return 30 * time.Minute
	case "h1", "1h":
		return time.Hour
	case "h4", "4h":
		return 4 * time.Hour
	case "d1", "1d":
		return 24 * time.Hour
	case "w1", "1w":
		return 7 * 24 * time.Hour
	default:
		// MN1 and unknown timeframes: best-effort (30 days)
		return 30 * 24 * time.Hour
	}
}

func alignOpenTime(t time.Time, tf string) time.Time {
	if t.IsZero() {
		return t
	}
	d := timeframeDuration(tf)
	if d <= 0 {
		return t
	}

	// Use UTC for deterministic alignment across services.
	u := t.UTC()
	sec := u.Unix()
	step := int64(d.Seconds())
	if step <= 0 {
		return u
	}
	aligned := (sec / step) * step
	return time.Unix(aligned, 0).UTC()
}

func alignKlineResponseTimes(k *KlineResponse) {
	if k == nil {
		return
	}
	openAt, ok := parseKlineTime(k.OpenTime)
	if !ok {
		return
	}
	openAt = alignOpenTime(openAt, k.Timeframe)
	closeAt := openAt.Add(timeframeDuration(k.Timeframe))
	k.OpenTime = openAt.Format("2006-01-02T15:04:05Z")
	k.CloseTime = closeAt.Format("2006-01-02T15:04:05Z")
}

func isTradingSessionOpen(t time.Time) bool {
	if t.IsZero() {
		return true
	}
	wd := t.UTC().Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	return true
}

func shouldAttemptRemoteBackfill(to time.Time, tf string) bool {
	// Minimal session awareness: skip remote backfill when market is closed (weekends).
	lastOpen := alignOpenTime(to, tf)
	return isTradingSessionOpen(lastOpen)
}

func klineBackfillWindow() time.Duration {
	v := strings.TrimSpace(os.Getenv("ANTRADER_KLINE_BACKFILL_WINDOW"))
	if v == "" {
		return 2 * time.Hour
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 2 * time.Hour
	}
	return d
}

func parseKlineTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func computeRange(req *KlineRequest) (time.Time, time.Time, int) {
	limit := req.Count
	if limit <= 0 {
		limit = 500
	}

	to := time.Now()
	if t, ok := parseKlineTime(req.To); ok {
		to = t
	}

	from := time.Time{}
	if t, ok := parseKlineTime(req.From); ok {
		from = t
	} else {
		// best-effort infer from count*timeframe for stable DB queries
		d := timeframeDuration(req.Timeframe)
		from = to.Add(-time.Duration(limit) * d)
	}

	return from, to, limit
}

func mergeKlines(a []*KlineResponse, b []*KlineResponse, limit int) []*KlineResponse {
	byOpen := make(map[string]*KlineResponse, len(a)+len(b))
	for _, k := range a {
		if k == nil || k.OpenTime == "" {
			continue
		}
		alignKlineResponseTimes(k)
		byOpen[k.OpenTime] = k
	}
	for _, k := range b {
		if k == nil || k.OpenTime == "" {
			continue
		}
		alignKlineResponseTimes(k)
		byOpen[k.OpenTime] = k
	}

	keys := make([]string, 0, len(byOpen))
	for k := range byOpen {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	res := make([]*KlineResponse, 0, len(keys))
	for _, k := range keys {
		res = append(res, byOpen[k])
	}
	if limit > 0 && len(res) > limit {
		res = res[len(res)-limit:]
	}
	return res
}

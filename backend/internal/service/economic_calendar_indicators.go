package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

// KeyIndicatorPoint represents a single historical data point for a macro
// indicator.
type KeyIndicatorPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// KeyIndicator represents a key macro indicator (e.g. CPI, unemployment rate)
// with its latest value and a small slice of historical observations for
// charting.
type KeyIndicator struct {
	Code        string              `json:"code"`
	SeriesID    string              `json:"seriesId"`
	Name        string              `json:"name"`
	Units       string              `json:"units"`
	Frequency   string              `json:"frequency"`
	LatestDate  string              `json:"latestDate"`
	LatestValue float64             `json:"latestValue"`
	History     []KeyIndicatorPoint `json:"history"`
}

// fredSeriesObservationsResponse models the response from
// fred/series/observations.
type fredSeriesObservationsResponse struct {
	Units        string `json:"units"`
	Count        int    `json:"count"`
	Observations []struct {
		Date  string `json:"date"`
		Value string `json:"value"`
	} `json:"observations"`
}

// GetKeyIndicators returns a predefined set of key macro indicators using
// local cache when possible and falling back to FRED's series/observations
// endpoint when necessary.
func (s *EconomicCalendarService) GetKeyIndicators(ctx context.Context) ([]*KeyIndicator, error) {
	if s == nil {
		return nil, fmt.Errorf("economic calendar service not initialized")
	}
	if s.apiKey == "" {
		return nil, fmt.Errorf("FRED API key is not configured")
	}
	if s.db == nil {
		return nil, fmt.Errorf("db is not configured for key indicators")
	}

	// Define the 4 default indicators.
	indicators := []struct {
		Code     string
		SeriesID string
		Name     string
	}{
		{"CPI", "CPIAUCSL", "CPI: All Items"},
		{"UNRATE", "UNRATE", "Unemployment Rate"},
		{"FEDFUNDS", "FEDFUNDS", "Effective Federal Funds Rate"},
		{"GDP", "GDPC1", "Real GDP"},
	}

	results := make([]*KeyIndicator, 0, len(indicators))

	for _, ind := range indicators {
		// First try to load recent history for this indicator from DB.
		ki, err := s.loadIndicatorFromDB(ctx, ind.Code, ind.SeriesID, ind.Name)
		if err == nil && ki != nil && len(ki.History) > 0 {
			results = append(results, ki)
			continue
		}

		// If DB has no data, fetch from FRED and persist.
		ki, err = s.fetchIndicatorFromFRED(ctx, ind.Code, ind.SeriesID, ind.Name)
		if err != nil {
			// Log and skip this indicator instead of failing the whole call.
			logger.Warn("failed to fetch key indicator from FRED",
				zap.String("code", ind.Code),
				zap.String("series_id", ind.SeriesID),
				zap.Error(err),
			)
			continue
		}
		results = append(results, ki)
	}

	return results, nil
}

// loadIndicatorFromDB loads recent observations for a given indicator from the
// macro_indicator_observations table.
func (s *EconomicCalendarService) loadIndicatorFromDB(ctx context.Context, code, seriesID, name string) (*KeyIndicator, error) {
	const query = `
SELECT
    obs_date,
    value
FROM macro_indicator_observations
WHERE indicator_code = $1
ORDER BY obs_date DESC
LIMIT 120`

	rows, err := s.db.QueryxContext(ctx, query, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]KeyIndicatorPoint, 0, 120)
	for rows.Next() {
		var (
			date  time.Time
			value *float64
		)
		if err := rows.Scan(&date, &value); err != nil {
			return nil, err
		}
		if value == nil {
			continue
		}
		history = append(history, KeyIndicatorPoint{
			Date:  date.Format("2006-01-02"),
			Value: *value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, nil
	}

	latest := history[0]
	return &KeyIndicator{
		Code:        code,
		SeriesID:    seriesID,
		Name:        name,
		Units:       "",
		Frequency:   "",
		LatestDate:  latest.Date,
		LatestValue: latest.Value,
		History:     history,
	}, nil
}

// fetchIndicatorFromFRED calls fred/series/observations for a given series ID,
// persists the observations, and returns a KeyIndicator value object.
func (s *EconomicCalendarService) fetchIndicatorFromFRED(ctx context.Context, code, seriesID, name string) (*KeyIndicator, error) {
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid FRED base URL: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/series/observations"

	values := u.Query()
	values.Set("api_key", s.apiKey)
	values.Set("file_type", "json")
	values.Set("series_id", seriesID)
	// Fetch the last 120 observations and let FRED handle ordering.
	values.Set("sort_order", "desc")
	values.Set("limit", "120")
	u.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call FRED series/observations API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		logger.Warn("FRED series/observations returned non-200",
			zap.String("status", resp.Status),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("FRED series/observations API returned status %s", resp.Status)
	}

	var raw fredSeriesObservationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode FRED series/observations response: %w", err)
	}
	if len(raw.Observations) == 0 {
		return nil, nil
	}

	// Persist observations into DB.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	stmt := `
INSERT INTO macro_indicator_observations (
    indicator_code,
    obs_date,
    value
)
VALUES ($1, $2, $3)
ON CONFLICT (indicator_code, obs_date) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = CURRENT_TIMESTAMP`

	history := make([]KeyIndicatorPoint, 0, len(raw.Observations))
	for _, obs := range raw.Observations {
		if obs.Value == "." || obs.Value == "" {
			continue
		}
		v, errParse := strconv.ParseFloat(obs.Value, 64)
		if errParse != nil {
			continue
		}
		// Parse date
		parsedDate, errDate := time.Parse("2006-01-02", obs.Date)
		if errDate != nil {
			continue
		}

		if _, errExec := tx.ExecContext(ctx, stmt, code, parsedDate, v); errExec != nil {
			err = errExec
			return nil, err
		}

		history = append(history, KeyIndicatorPoint{
			Date:  parsedDate.Format("2006-01-02"),
			Value: v,
		})
	}
	if len(history) == 0 {
		return nil, nil
	}

	// history currently is newest-first because we requested sort_order=desc.
	latest := history[0]
	return &KeyIndicator{
		Code:        code,
		SeriesID:    seriesID,
		Name:        name,
		Units:       raw.Units,
		Frequency:   "",
		LatestDate:  latest.Date,
		LatestValue: latest.Value,
		History:     history,
	}, nil
}

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

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"anttrader/internal/config"
	"anttrader/pkg/logger"
)

type EconomicCalendarService struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	db         *sqlx.DB
	dynCfg     *DynamicConfigService
}

type EconomicCalendarQuery struct {
	From       string
	To         string
	Country    string
	Symbol     string
	Importance string
	Lang       string
}

type EconomicCalendarEvent struct {
	Date    string `json:"date" db:"date"`
	Time    string `json:"time,omitempty" db:"time"`
	Country string `json:"country,omitempty" db:"country"`
	Event   string `json:"event,omitempty" db:"event"`
	// LocalizedEvent is an optional, translated version of Event for the
	// requested UI language. It is not stored in the economic_events table and
	// is populated at read time using the event_translations table + LLM
	// translation.
	LocalizedEvent string `json:"localizedEvent,omitempty" db:"-"`
	Impact         string `json:"impact,omitempty" db:"impact"`
	Actual         string `json:"actual,omitempty" db:"actual"`
	Previous       string `json:"previous,omitempty" db:"previous"`
	Estimate       string `json:"estimate,omitempty" db:"estimate"`
	Unit           string `json:"unit,omitempty" db:"unit"`
	Currency       string `json:"currency,omitempty" db:"currency"`
	Timestamp      int64  `json:"timestamp,omitempty" db:"timestamp"`
}

// fredReleasesDatesResponse models the response from FRED fred/releases/dates endpoint.
type fredReleasesDatesResponse struct {
	ReleaseDates []struct {
		ReleaseID   int    `json:"release_id"`
		ReleaseName string `json:"release_name"`
		Date        string `json:"date"`
	} `json:"release_dates"`
}

// defaultTranslationLangs defines which UI languages we eagerly translate
// economic calendar event titles for after fetching from the external API.
// 'en' 使用英文原文，因此不需要翻译。
var defaultTranslationLangs = []string{"zh-cn", "zh-tw", "ja", "vi"}

func NewEconomicCalendarService(cfg *config.FMPConfig, db *sqlx.DB, dynCfg *DynamicConfigService) *EconomicCalendarService {
	client := &http.Client{Timeout: 10 * time.Second}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		// Default to FRED API base URL
		baseURL = "https://api.stlouisfed.org/fred"
	}
	return &EconomicCalendarService{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: client,
		db:         db,
		dynCfg:     dynCfg,
	}
}

func (s *EconomicCalendarService) GetCalendar(ctx context.Context, q *EconomicCalendarQuery) ([]*EconomicCalendarEvent, error) {
	if s == nil {
		return nil, fmt.Errorf("economic calendar service not initialized")
	}
	if s.apiKey == "" {
		return nil, fmt.Errorf("FRED API key is not configured")
	}

	// 1) Try to load events from local database first. This avoids hitting the
	// external API on every request and gives us a persistent history.
	var result []*EconomicCalendarEvent
	if s.db != nil {
		if events, err := s.loadFromDB(ctx, q); err == nil && len(events) > 0 {
			result = events
		}
	}

	// 2) If DB has no events yet, fall back to FRED API.
	if result == nil {
		u, err := url.Parse(s.baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid FMP base URL: %w", err)
		}
		// Use FRED fred/releases/dates endpoint to approximate an economic calendar of
		// recent and upcoming macroeconomic releases.
		u.Path = strings.TrimRight(u.Path, "/") + "/releases/dates"

		values := u.Query()
		values.Set("api_key", s.apiKey)
		values.Set("file_type", "json")
		// We fetch the most recent 100 release dates ordered by release_date desc.
		values.Set("limit", strconv.Itoa(100))
		values.Set("order_by", "release_date")
		values.Set("sort_order", "desc")
		// We keep the default include_release_dates_with_no_data=false so that
		// releases without associated data (often future-only calendar entries) are
		// excluded. This keeps the list focused on releases with data.
		// NOTE: Any optional filters from EconomicCalendarQuery (from/to/country/etc)
		// are currently ignored because the FRED endpoint does not support them
		// directly, but the method retains the parameter for future extension.
		u.RawQuery = values.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to call FRED releases/dates API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			logger.Warn("FRED releases/dates returned non-200",
				zap.String("status", resp.Status),
				zap.String("body", string(body)),
			)
			return nil, fmt.Errorf("FRED releases/dates API returned status %s", resp.Status)
		}

		var raw fredReleasesDatesResponse
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return nil, fmt.Errorf("failed to decode FRED releases/dates response: %w", err)
		}

		events := make([]*EconomicCalendarEvent, 0, len(raw.ReleaseDates))
		for _, d := range raw.ReleaseDates {
			// Parse date to Unix timestamp if possible.
			var ts int64
			if d.Date != "" {
				if tParsed, err := time.Parse("2006-01-02", d.Date); err == nil {
					ts = tParsed.Unix()
				}
			}

			events = append(events, &EconomicCalendarEvent{
				Date:      d.Date,
				Country:   "US", // FRED is primarily U.S. data; adjust if you later add other regions.
				Event:     d.ReleaseName,
				Timestamp: ts,
			})
		}
		result = events

		// 3) Persist fetched events to database for future reads. We swallow errors
		// here so that a write failure does not break the API response.
		if s.db != nil {
			if err := s.saveToDB(ctx, result); err != nil {
				logger.Warn("failed to persist economic events", zap.Error(err))
			}
		}

		// 4) After we have the latest events persisted, proactively translate
		// titles for a small set of default languages so subsequent API calls for
		// those languages only need to read from the database.
		for _, lng := range defaultTranslationLangs {
			// ignore errors here; they will be retried lazily on demand via
			// localizeEvents when that language is requested.
			if err := s.localizeEvents(ctx, result, lng); err != nil {
				logger.Warn("failed to prewarm economic event translations",
					zap.String("lang", lng),
					zap.Error(err),
				)
			}
		}
	}

	// 5) Optionally translate event titles for the requested language using LLM
	// with database caching. In most cases, translations will already exist from
	// the prewarm step above, so this call will only read from the cache.
	lang := normalizeLang(q.Lang)
	if lang != "" && lang != "en" {
		if err := s.localizeEvents(ctx, result, lang); err != nil {
			logger.Warn("failed to localize economic events",
				zap.String("lang", lang),
				zap.Error(err),
			)
		}
	}

	return result, nil
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

// loadFromDB loads previously persisted economic events from the local
// economic_events table. For now we keep the logic simple: we return the most
// recent 100 events ordered by date descending. Query parameters are accepted
// but currently ignored; they are kept for future extension.
func (s *EconomicCalendarService) loadFromDB(ctx context.Context, _ *EconomicCalendarQuery) ([]*EconomicCalendarEvent, error) {
	if s.db == nil {
		return nil, fmt.Errorf("db is not configured for economic calendar service")
	}

	var events []*EconomicCalendarEvent
	const query = `
SELECT
    date,
    time,
    country,
    event,
    impact,
    actual,
    previous,
    estimate,
    unit,
    currency,
    COALESCE(timestamp, 0) AS timestamp
FROM economic_events
WHERE source = 'FRED_RELEASES_DATES'
ORDER BY date DESC, created_at DESC
LIMIT 100`

	if err := s.db.SelectContext(ctx, &events, query); err != nil {
		return nil, err
	}
	return events, nil
}

// saveToDB persists fetched economic events into the economic_events table.
// It performs simple upserts keyed by (source, external_id, date) where
// external_id is the FRED release_id rendered as a string.
func (s *EconomicCalendarService) saveToDB(ctx context.Context, events []*EconomicCalendarEvent) error {
	if s.db == nil {
		return fmt.Errorf("db is not configured for economic calendar service")
	}
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		} else {
			_ = tx.Commit()
		}
	}()

	stmt := `
INSERT INTO economic_events (
    source,
    external_id,
    date,
    time,
    country,
    event,
    impact,
    actual,
    previous,
    estimate,
    unit,
    currency,
    timestamp
)
VALUES (
    'FRED_RELEASES_DATES',
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13
)
ON CONFLICT (source, external_id, date) DO UPDATE SET
    time = EXCLUDED.time,
    country = EXCLUDED.country,
    event = EXCLUDED.event,
    impact = EXCLUDED.impact,
    actual = EXCLUDED.actual,
    previous = EXCLUDED.previous,
    estimate = EXCLUDED.estimate,
    unit = EXCLUDED.unit,
    currency = EXCLUDED.currency,
    timestamp = EXCLUDED.timestamp,
    updated_at = CURRENT_TIMESTAMP`

	for _, e := range events {
		// For FRED releases/dates we don't have a natural external id in the
		// mapped struct, so we use the date + event name as a synthetic external
		// identifier to keep rows stable.
		externalID := fmt.Sprintf("%s|%s", e.Date, e.Event)
		if _, errExec := tx.ExecContext(ctx, stmt,
			externalID,
			e.Date,
			e.Time,
			e.Country,
			e.Event,
			e.Impact,
			e.Actual,
			e.Previous,
			e.Estimate,
			e.Unit,
			e.Currency,
			e.Timestamp,
		); errExec != nil {
			err = errExec
			return err
		}
	}

	return nil
}

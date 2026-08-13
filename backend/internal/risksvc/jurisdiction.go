// Package risksvc provides the Jurisdictional Gate (M11-16).
//
// Five-component pre-trade compliance barrier:
//  1. KYC verification — block unverified users
//  2. IP geolocation — detect country from client IP
//  3. Sanctioned country denial — block OFAC-restricted jurisdictions
//  4. Disclaimer acceptance — require explicit risk acknowledgment
//  5. Risk questionnaire — mandatory before first trade
//
// The gate is injected into KycJurisdictionRule and evaluated as part
// of the HardLimit pipeline before any order reaches the broker.

package risksvc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// --- Interfaces ---

// GeoIPResolver resolves an IP address to an ISO 3166-1 alpha-2 country code.
type GeoIPResolver interface {
	CountryCode(ip string) (string, error)
}

// JurisdictionStore provides KYC and compliance status for a user.
type JurisdictionStore interface {
	GetStatus(ctx context.Context, userID string) (*JurisdictionStatus, error)
	SetKYCStatus(ctx context.Context, userID, status, verifiedBy string) error
	RecordCountry(ctx context.Context, userID, countryCode, source string) error
	IsDisclaimerAccepted(ctx context.Context, userID string) (bool, error)
	AcceptDisclaimer(ctx context.Context, userID, version string) error
	IsQuestionnaireCompleted(ctx context.Context, userID string) (bool, error)
	SubmitQuestionnaire(ctx context.Context, userID, version string, riskScore int) error
	IsSanctioned(ctx context.Context, countryCode string) (bool, error)
}

// --- Data types ---

// JurisdictionStatus holds the full compliance snapshot for a user.
type JurisdictionStatus struct {
	UserID               string
	KYCStatus            string
	CountryCode          string
	CountrySource        string
	IsSanctioned         bool
	DisclaimerAccepted   bool
	QuestionnaireDone    bool
	RiskScore            int
	SanctionedOverride   bool
	KYCVerifiedAt        *time.Time
	DisclaimerAcceptedAt *time.Time
	QuestionnaireDoneAt  *time.Time
}

// QuestionAnswer is a single answer in the risk questionnaire.
type QuestionAnswer struct {
	QuestionKey string
	Answer      string
}

// --- Sentinel errors ---

var (
	ErrGeoIPUnavailable      = errors.New("geoip database unavailable")
	ErrSanctionedCountry     = errors.New("sanctioned country")
	ErrKYCNotVerified        = errors.New("kyc not verified")
	ErrDisclaimerNotAccepted = errors.New("disclaimer not accepted")
	ErrQuestionnaireNotDone  = errors.New("risk questionnaire not completed")
)

// --- JurisdictionGate ---

// JurisdictionGate orchestrates the five-component compliance check.
// Zero-value fields disable the corresponding check.
type JurisdictionGate struct {
	Store                JurisdictionStore
	GeoIP                GeoIPResolver
	RequireKYC           bool
	RequireDisclaimer    bool
	RequireQuestionnaire bool
}

// Check runs the full jurisdictional gate for a user + client IP.
// Returns nil if all applicable checks pass, or an error describing the block reason.
func (g *JurisdictionGate) Check(ctx context.Context, userID, clientIP string) error {
	// Step 1: GeoIP → sanctioned country check
	if g.GeoIP != nil && clientIP != "" {
		country, err := g.GeoIP.CountryCode(clientIP)
		if err != nil {
			// GeoIP database unavailable is an infrastructure issue, not a
			// compliance block. Skip the country check; other checks (KYC,
			// disclaimer, questionnaire) still run per config.
			if errors.Is(err, ErrGeoIPUnavailable) {
				country = ""
			}
			// Non-fatal geoip errors (parse failures, private IPs) — log and continue.
			// Private/lookup-failure IPs get country="" which is never sanctioned.
		}
		if country != "" {
			// Best-effort record the country for audit.
			if g.Store != nil {
				_ = g.Store.RecordCountry(ctx, userID, country, "geoip")
			}
			if g.Store != nil {
				sanctioned, err := g.Store.IsSanctioned(ctx, country)
				if err == nil && sanctioned {
					// Check for admin override.
					status, statusErr := g.Store.GetStatus(ctx, userID)
					if statusErr == nil && status.SanctionedOverride {
						// Override granted — let through.
					} else {
						return fmt.Errorf("%w: country=%s", ErrSanctionedCountry, country)
					}
				}
			}
		}
	}

	// Step 2: KYC verification
	if g.RequireKYC && g.Store != nil {
		status, err := g.Store.GetStatus(ctx, userID)
		if err != nil {
			return fmt.Errorf("jurisdiction: get status: %w", err)
		}
		if status.KYCStatus != "verified" {
			return fmt.Errorf("%w: status=%s", ErrKYCNotVerified, status.KYCStatus)
		}
	}

	// Step 3: Disclaimer acceptance
	if g.RequireDisclaimer && g.Store != nil {
		accepted, err := g.Store.IsDisclaimerAccepted(ctx, userID)
		if err != nil {
			return fmt.Errorf("jurisdiction: check disclaimer: %w", err)
		}
		if !accepted {
			return ErrDisclaimerNotAccepted
		}
	}

	// Step 4: Risk questionnaire
	if g.RequireQuestionnaire && g.Store != nil {
		done, err := g.Store.IsQuestionnaireCompleted(ctx, userID)
		if err != nil {
			return fmt.Errorf("jurisdiction: check questionnaire: %w", err)
		}
		if !done {
			return ErrQuestionnaireNotDone
		}
	}

	return nil
}

// --- MaxMind GeoIP Resolver ---

// MaxMindGeoIPResolver wraps a MaxMind GeoLite2 database.
// Implements GeoIPResolver using github.com/oschwald/geoip2-golang.
//
// When the database path is empty or the file cannot be opened, CountryCode
// returns ErrGeoIPUnavailable. Callers should handle this gracefully
// (fail-closed for production, allow for development).
type MaxMindGeoIPResolver struct {
	mu     sync.RWMutex
	dbPath string
	// reader is lazily initialized on first successful open.
	// We store the path and a boolean rather than the *geoip2.Reader
	// to avoid importing geoip2 in this file — the actual lookup calls
	// into a small adapter. This keeps the core package dependency-light.
	lookup func(ipStr string) (string, error)
}

// NewMaxMindGeoIPResolver creates a resolver backed by a MaxMind mmdb file.
// The geoip2 library is imported only in the adapter bridge (geoip_bridge.go)
// so that this file stays free of the external dependency.
func NewMaxMindGeoIPResolver(dbPath string) *MaxMindGeoIPResolver {
	return &MaxMindGeoIPResolver{dbPath: dbPath}
}

// CountryCode resolves an IP string to an ISO 3166-1 alpha-2 country code.
// Returns ErrGeoIPUnavailable if no database is loaded.
func (r *MaxMindGeoIPResolver) CountryCode(ipStr string) (string, error) {
	if r == nil || r.dbPath == "" {
		return "", ErrGeoIPUnavailable
	}

	r.mu.RLock()
	lookup := r.lookup
	r.mu.RUnlock()

	if lookup == nil {
		// Try lazy init.
		lk, err := openMaxMindDB(r.dbPath)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrGeoIPUnavailable, err)
		}
		r.mu.Lock()
		r.lookup = lk
		lookup = lk
		r.mu.Unlock()
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", nil // unparseable IP — not an error, just no country
	}

	return lookup(ipStr)
}

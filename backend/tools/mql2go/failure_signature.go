package mql2go

import (
	antv1 "alphaforge/gen/proto/ant/v1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// FailureSignature is a deduplication key for backtest failures (§12.1 Layer 2).
// Two EAs with the same failure signature have the same root cause and only
// need to be fixed once.
type FailureSignature struct {
	Hash        string   // SHA256 of (sourceHash + ruleIDs + blindSpotIDs)
	SourceHash  string   // SHA256 of anonymized EA source
	RuleIDs     []string // matched rule IDs (e.g. R01, R05)
	BlindSpots  []string // blind spot builtin names
	TotalTrades int      // backtest trade count
	CreatedAt   time.Time
}

// ReproPackage is a self-contained reproduction package for offline analysis.
// It contains everything needed to reproduce and diagnose a failure without
// access to the original user's data.
type ReproPackage struct {
	Signature     FailureSignature
	SourceHash    string // SHA256 of raw source (for dedup)
	SourcePreview string // first 500 chars of source (for human review)
	Findings      []DiagnosticFinding
	BlindSpots    []CoverageBlindSpot
	RuntimeBlinds []RuntimeBlindSpot
	TotalTrades   int
	Symbol        string
	Timeframe     string
	CreatedAt     time.Time
}

// hashSource computes a SHA256 hash of the EA source.
// The source is normalized (whitespace trimmed) to dedup similar EAs.
func hashSource(source string) string {
	normalized := strings.TrimSpace(source)
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}

// BuildFailureSignature creates a dedup signature from rule findings and blind spots.
func BuildFailureSignature(source string, findings []DiagnosticFinding, blindSpots []CoverageBlindSpot, runtimeBlinds []RuntimeBlindSpot, totalTrades int) FailureSignature {
	return BuildFailureSignatureAt(source, findings, blindSpots, runtimeBlinds, totalTrades, time.Now().UTC())
}

// BuildReproPackage creates a full reproduction package.
func BuildReproPackage(source string, findings []DiagnosticFinding, blindSpots []CoverageBlindSpot, runtimeBlinds []RuntimeBlindSpot, totalTrades int, symbol string, timeframe string) ReproPackage {
	now := time.Now().UTC()
	sig := BuildFailureSignatureAt(source, findings, blindSpots, runtimeBlinds, totalTrades, now)
	preview := source
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	return ReproPackage{
		Signature:     sig,
		SourceHash:    sig.SourceHash,
		SourcePreview: preview,
		Findings:      findings,
		BlindSpots:    blindSpots,
		RuntimeBlinds: runtimeBlinds,
		TotalTrades:   totalTrades,
		Symbol:        symbol,
		Timeframe:     timeframe,
		CreatedAt:     now,
	}
}

// BuildFailureSignatureAt is like BuildFailureSignature but accepts a timestamp.
// Used by BuildReproPackage to ensure Signature.CreatedAt == ReproPackage.CreatedAt.
func BuildFailureSignatureAt(source string, findings []DiagnosticFinding, blindSpots []CoverageBlindSpot, runtimeBlinds []RuntimeBlindSpot, totalTrades int, now time.Time) FailureSignature {
	sourceHash := hashSource(source)

	var ruleIDs []string
	for _, f := range findings {
		ruleIDs = append(ruleIDs, f.RuleID)
	}

	bsSeen := make(map[string]bool)
	var bsNames []string
	for _, bs := range blindSpots {
		if !bsSeen[bs.Builtin] {
			bsSeen[bs.Builtin] = true
			bsNames = append(bsNames, bs.Builtin)
		}
	}
	for _, rbs := range runtimeBlinds {
		if !bsSeen[rbs.Builtin] {
			bsSeen[rbs.Builtin] = true
			bsNames = append(bsNames, rbs.Builtin)
		}
	}

	sort.Strings(ruleIDs)
	sort.Strings(bsNames)
	hashInput := sourceHash + "|" + strings.Join(ruleIDs, ",") + "|" + strings.Join(bsNames, ",")
	h := sha256.Sum256([]byte(hashInput))

	return FailureSignature{
		Hash:        hex.EncodeToString(h[:]),
		SourceHash:  sourceHash,
		RuleIDs:     ruleIDs,
		BlindSpots:  bsNames,
		TotalTrades: totalTrades,
		CreatedAt:   now,
	}
}

// String returns a human-readable summary of the failure signature.
func (f FailureSignature) String() string {
	shortHash := f.Hash
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}
	return fmt.Sprintf("FailureSignature{hash=%s, rules=%v, blindSpots=%v, trades=%d}",
		shortHash, f.RuleIDs, f.BlindSpots, f.TotalTrades)
}

// ToProto converts the Go FailureSignature to its proto representation.
func (f FailureSignature) ToProto() *antv1.FailureSignature {
	return &antv1.FailureSignature{
		Hash:        f.Hash,
		SourceHash:  f.SourceHash,
		RuleIds:     f.RuleIDs,
		BlindSpots:  f.BlindSpots,
		TotalTrades: int32(f.TotalTrades),
		CreatedAtMs: f.CreatedAt.UnixMilli(),
	}
}

// ToProto converts the Go ReproPackage to its proto representation.
func (r ReproPackage) ToProto() *antv1.ReproPackage {
	findings := make([]*antv1.DiagnosticFinding, len(r.Findings))
	for i, f := range r.Findings {
		findings[i] = &antv1.DiagnosticFinding{
			RuleId:   f.RuleID,
			Severity: f.Severity,
			Title:    f.Title,
			Detail:   f.Detail,
			Suggest:  f.Suggest,
		}
	}
	return &antv1.ReproPackage{
		Signature:     r.Signature.ToProto(),
		SourceHash:    r.SourceHash,
		SourcePreview: r.SourcePreview,
		Findings:      findings,
		TotalTrades:   int32(r.TotalTrades),
		Symbol:        r.Symbol,
		Timeframe:     r.Timeframe,
		CreatedAtMs:   r.CreatedAt.UnixMilli(),
	}
}

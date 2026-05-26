package qualitygate

import "math"

// QualityScore is a detailed breakdown of strategy quality.
type QualityScore struct {
	Overall    float64           `json:"overall"`    // 0–100 weighted composite
	RiskLevel  string            `json:"risk_level"` // low/medium/high
	Components map[string]float64 `json:"components"` // per-gate scores
	Verdict    string            `json:"verdict"`     // APPROVED / NEEDS_REVIEW / REJECTED
	IsReliable bool              `json:"is_reliable"`
}

// ComputeQualityScore derives a comprehensive quality score from pipeline results.
func ComputeQualityScore(pr *PipelineResult) *QualityScore {
	qs := &QualityScore{
		Overall:    math.Round(pr.Score*10000) / 100, // 0–100 with 2 decimals
		RiskLevel:  pr.RiskLevel,
		IsReliable: pr.Passed && pr.Score >= 0.6,
		Components: make(map[string]float64),
	}

	for _, r := range pr.Results {
		qs.Components[r.Gate] = math.Round(r.Score*100) / 100
	}

	// Verdict logic
	criticalCount := 0
	errorCount := 0
	for _, r := range pr.Results {
		switch r.Severity {
		case SeverityCritical:
			criticalCount++
		case SeverityError:
			errorCount++
		}
	}

	switch {
	case criticalCount > 0:
		qs.Verdict = "REJECTED"
	case errorCount > 0:
		qs.Verdict = "NEEDS_REVIEW"
	case pr.Score >= 0.7:
		qs.Verdict = "APPROVED"
	case pr.Score >= 0.5:
		qs.Verdict = "NEEDS_REVIEW"
	default:
		qs.Verdict = "REJECTED"
	}

	return qs
}

// ApprovalDecision wraps quality scores with a final go/no-go decision.
type ApprovalDecision struct {
	Allowed     bool          `json:"allowed"`
	Quality     *QualityScore `json:"quality"`
	Reason      string        `json:"reason"`
	Requires    []string      `json:"requires"` // what must be fixed
}

// DecideApproval makes the final approval decision based on quality scores.
func DecideApproval(pr *PipelineResult) *ApprovalDecision {
	qs := ComputeQualityScore(pr)
	ad := &ApprovalDecision{
		Quality:  qs,
		Requires: make([]string, 0),
	}

	switch qs.Verdict {
	case "APPROVED":
		ad.Allowed = true
		ad.Reason = "all quality gates passed with sufficient scores"
	case "REJECTED":
		ad.Allowed = false
		ad.Reason = "one or more critical quality gates failed"
		for _, err := range pr.Errors {
			ad.Requires = append(ad.Requires, err)
		}
	case "NEEDS_REVIEW":
		ad.Allowed = false
		ad.Reason = "quality gates require manual review"
		for _, w := range pr.Warnings {
			ad.Requires = append(ad.Requires, "WARNING: "+w)
		}
	}

	return ad
}

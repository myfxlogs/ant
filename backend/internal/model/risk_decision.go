package model

const (
	RiskDecisionAllow  = "allow"
	RiskDecisionReject = "reject"

	RiskDecisionSourceManual   = "manual"
	RiskDecisionSourceAuto     = "auto"
	RiskDecisionSourceSchedule = "schedule"
)

type RiskDecision struct {
	Decision  string                 `json:"decision"`
	Allowed   bool                   `json:"allowed"`
	Source    string                 `json:"source"`
	Code      string                 `json:"code,omitempty"`
	Reason    string                 `json:"reason,omitempty"`
	Retryable bool                   `json:"retryable"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

func NewRiskDecision(allowed bool, source, code, reason string, retryable bool) *RiskDecision {
	decision := RiskDecisionReject
	if allowed {
		decision = RiskDecisionAllow
	}
	return &RiskDecision{Decision: decision, Allowed: allowed, Source: source, Code: code, Reason: reason, Retryable: retryable}
}

func AllowRiskDecision(source string) *RiskDecision {
	return NewRiskDecision(true, source, "OK", "", false)
}

func RejectRiskDecision(source, code, reason string, retryable bool) *RiskDecision {
	return NewRiskDecision(false, source, code, reason, retryable)
}

func (r *RiskCheckResult) SetDecision(decision *RiskDecision) {
	if r == nil || decision == nil {
		return
	}
	r.Decision = decision
	r.Allowed = decision.Allowed
	r.IsWithinLimits = decision.Allowed
	if decision.Reason != "" {
		r.Reason = decision.Reason
	}
}

// Package risksvc — stub implementations for testing.
package risksvc

import "context"

// StubGeoIPResolver is a test-only resolver that maps IP strings to country codes.
type StubGeoIPResolver struct {
	Countries map[string]string
	Err       error
}

func (r *StubGeoIPResolver) CountryCode(ip string) (string, error) {
	if r.Err != nil {
		return "", r.Err
	}
	if r.Countries == nil {
		return "", nil
	}
	c, ok := r.Countries[ip]
	if !ok {
		return "", nil
	}
	return c, nil
}

// StubJurisdictionStore is an in-memory JurisdictionStore for tests.
type StubJurisdictionStore struct {
	Statuses            map[string]*JurisdictionStatus
	SanctionedCountries map[string]bool
	Disclaimers         map[string]bool
	Questionnaires      map[string]bool
}

func NewStubJurisdictionStore() *StubJurisdictionStore {
	return &StubJurisdictionStore{
		Statuses:            make(map[string]*JurisdictionStatus),
		SanctionedCountries: make(map[string]bool),
		Disclaimers:         make(map[string]bool),
		Questionnaires:      make(map[string]bool),
	}
}

func (s *StubJurisdictionStore) GetStatus(_ context.Context, userID string) (*JurisdictionStatus, error) {
	if st, ok := s.Statuses[userID]; ok {
		return st, nil
	}
	return &JurisdictionStatus{UserID: userID, KYCStatus: "unverified"}, nil
}

func (s *StubJurisdictionStore) SetKYCStatus(_ context.Context, userID, status, _ string) error {
	st, ok := s.Statuses[userID]
	if !ok {
		st = &JurisdictionStatus{UserID: userID}
		s.Statuses[userID] = st
	}
	st.KYCStatus = status
	return nil
}

func (s *StubJurisdictionStore) RecordCountry(_ context.Context, userID, code, source string) error {
	st, ok := s.Statuses[userID]
	if !ok {
		st = &JurisdictionStatus{UserID: userID}
		s.Statuses[userID] = st
	}
	st.CountryCode = code
	st.CountrySource = source
	return nil
}

func (s *StubJurisdictionStore) IsDisclaimerAccepted(_ context.Context, userID string) (bool, error) {
	return s.Disclaimers[userID], nil
}

func (s *StubJurisdictionStore) AcceptDisclaimer(_ context.Context, userID, _ string) error {
	s.Disclaimers[userID] = true
	return nil
}

func (s *StubJurisdictionStore) IsQuestionnaireCompleted(_ context.Context, userID string) (bool, error) {
	return s.Questionnaires[userID], nil
}

func (s *StubJurisdictionStore) SubmitQuestionnaire(_ context.Context, userID, _ string, score int) error {
	s.Questionnaires[userID] = true
	if st, ok := s.Statuses[userID]; ok {
		st.RiskScore = score
		st.QuestionnaireDone = true
	}
	return nil
}

func (s *StubJurisdictionStore) IsSanctioned(_ context.Context, countryCode string) (bool, error) {
	return s.SanctionedCountries[countryCode], nil
}

package risksvc

import (
	"context"
	"errors"
	"testing"
)

func TestJurisdictionGate(t *testing.T) {
	t.Run("all clear with verified KYC and clean IP", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-1", "verified", "")
		store.Disclaimers["user-1"] = true
		store.Questionnaires["user-1"] = true

		geo := &StubGeoIPResolver{Countries: map[string]string{"1.2.3.4": "US"}}

		gate := &JurisdictionGate{
			Store:               store,
			GeoIP:               geo,
			RequireKYC:          true,
			RequireDisclaimer:   true,
			RequireQuestionnaire: true,
		}

		err := gate.Check(context.Background(), "user-1", "1.2.3.4")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("block sanctioned country", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SanctionedCountries["IR"] = true

		geo := &StubGeoIPResolver{Countries: map[string]string{"5.6.7.8": "IR"}}

		gate := &JurisdictionGate{
			Store: store,
			GeoIP: geo,
		}

		err := gate.Check(context.Background(), "user-2", "5.6.7.8")
		if err == nil {
			t.Fatal("expected error for sanctioned country, got nil")
		}
		if !errors.Is(err, ErrSanctionedCountry) {
			t.Fatalf("expected ErrSanctionedCountry, got %v", err)
		}
	})

	t.Run("allow sanctioned country with admin override", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SanctionedCountries["IR"] = true
		store.Statuses["user-3"] = &JurisdictionStatus{
			UserID:             "user-3",
			KYCStatus:          "verified",
			SanctionedOverride: true,
		}

		geo := &StubGeoIPResolver{Countries: map[string]string{"5.6.7.8": "IR"}}

		gate := &JurisdictionGate{
			Store: store,
			GeoIP: geo,
		}

		err := gate.Check(context.Background(), "user-3", "5.6.7.8")
		if err != nil {
			t.Fatalf("expected nil (override), got %v", err)
		}
	})

	t.Run("block unverified KYC", func(t *testing.T) {
		store := NewStubJurisdictionStore()

		gate := &JurisdictionGate{
			Store:      store,
			RequireKYC: true,
		}

		err := gate.Check(context.Background(), "user-4", "")
		if err == nil {
			t.Fatal("expected KYC error, got nil")
		}
		if !errors.Is(err, ErrKYCNotVerified) {
			t.Fatalf("expected ErrKYCNotVerified, got %v", err)
		}
	})

	t.Run("block missing disclaimer", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-5", "verified", "")

		gate := &JurisdictionGate{
			Store:             store,
			RequireKYC:        true,
			RequireDisclaimer: true,
		}

		err := gate.Check(context.Background(), "user-5", "")
		if err == nil {
			t.Fatal("expected disclaimer error, got nil")
		}
		if !errors.Is(err, ErrDisclaimerNotAccepted) {
			t.Fatalf("expected ErrDisclaimerNotAccepted, got %v", err)
		}
	})

	t.Run("block missing questionnaire", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-6", "verified", "")
		store.Disclaimers["user-6"] = true

		gate := &JurisdictionGate{
			Store:                store,
			RequireKYC:          true,
			RequireDisclaimer:   true,
			RequireQuestionnaire: true,
		}

		err := gate.Check(context.Background(), "user-6", "")
		if err == nil {
			t.Fatal("expected questionnaire error, got nil")
		}
		if !errors.Is(err, ErrQuestionnaireNotDone) {
			t.Fatalf("expected ErrQuestionnaireNotDone, got %v", err)
		}
	})

	t.Run("geoip unavailable — fail-closed", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		geo := &StubGeoIPResolver{Err: ErrGeoIPUnavailable}

		gate := &JurisdictionGate{
			Store: store,
			GeoIP: geo,
		}

		err := gate.Check(context.Background(), "user-7", "1.2.3.4")
		if err == nil {
			t.Fatal("expected geoip unavailable error, got nil")
		}
		if !errors.Is(err, ErrGeoIPUnavailable) {
			t.Fatalf("expected ErrGeoIPUnavailable, got %v", err)
		}
	})

	t.Run("no client IP — skip geoip, check KYC only", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-8", "verified", "")

		gate := &JurisdictionGate{
			Store:      store,
			RequireKYC: true,
		}

		err := gate.Check(context.Background(), "user-8", "")
		if err != nil {
			t.Fatalf("expected nil (no IP, KYC ok), got %v", err)
		}
	})

	t.Run("non-sanctioned country with clean status", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-9", "verified", "")
		store.Disclaimers["user-9"] = true
		store.Questionnaires["user-9"] = true

		geo := &StubGeoIPResolver{Countries: map[string]string{"2.3.4.5": "JP"}}

		gate := &JurisdictionGate{
			Store:               store,
			GeoIP:               geo,
			RequireKYC:          true,
			RequireDisclaimer:   true,
			RequireQuestionnaire: true,
		}

		err := gate.Check(context.Background(), "user-9", "2.3.4.5")
		if err != nil {
			t.Fatalf("expected nil for JP, got %v", err)
		}
	})

	t.Run("all gates disabled — pass through", func(t *testing.T) {
		gate := &JurisdictionGate{}
		err := gate.Check(context.Background(), "anyone", "")
		if err != nil {
			t.Fatalf("expected nil (no gates), got %v", err)
		}
	})

	t.Run("KYC rejected status blocks", func(t *testing.T) {
		store := NewStubJurisdictionStore()
		store.SetKYCStatus(context.Background(), "user-10", "rejected", "")

		gate := &JurisdictionGate{
			Store:      store,
			RequireKYC: true,
		}

		err := gate.Check(context.Background(), "user-10", "")
		if err == nil {
			t.Fatal("expected KYC error for rejected status, got nil")
		}
	})
}

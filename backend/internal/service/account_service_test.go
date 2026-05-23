package service

import (
	"testing"
)

func TestParseHostPort(t *testing.T) {
	svc := &AccountService{}

	tests := []struct {
		input      string
		expectHost string
		expectPort int32
	}{
		{"192.168.1.1:443", "192.168.1.1", 443},
		{"example.com:443", "example.com", 443},
		{"example.com:8080", "example.com", 8080},
		{"example.com", "example.com", 443},
		{"", "", 443},
		{"mt4.example.com:443", "mt4.example.com", 443},
	}

	for _, tt := range tests {
		host, port := svc.parseHostPort(tt.input)
		if host != tt.expectHost || port != tt.expectPort {
			t.Errorf("parseHostPort(%s) = (%s, %d), expected (%s, %d)",
				tt.input, host, port, tt.expectHost, tt.expectPort)
		}
	}
}

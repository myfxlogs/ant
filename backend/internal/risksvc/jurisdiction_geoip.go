package risksvc

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

// openMaxMindDB opens a MaxMind GeoLite2 database and returns a lookup function.
// The returned function maps an IP string to an ISO country code.
func openMaxMindDB(dbPath string) (func(ipStr string) (string, error), error) {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return nil, err
	}

	return func(ipStr string) (string, error) {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return "", nil
		}
		record, err := db.Country(ip)
		if err != nil {
			return "", nil
		}
		return record.Country.IsoCode, nil
	}, nil
}

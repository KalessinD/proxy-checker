package common

import (
	"fmt"
	"net"
	"proxy-checker/internal/common/i18n"

	"github.com/oschwald/maxminddb-golang"
)

type GeoIPResolver interface {
	ResolveCountry(ip string) string
	Close() error
}

type MaxMindDBResolver struct {
	db *maxminddb.Reader
}

func NewMaxMindDBResolverFromBytes(data []byte) (GeoIPResolver, error) {
	db, err := maxminddb.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("geoip.err_open"), err)
	}
	return &MaxMindDBResolver{db: db}, nil
}

func NewMaxMindDBResolverFromFile(filePath string) (GeoIPResolver, error) {
	db, err := maxminddb.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("geoip.err_open"), err)
	}
	return &MaxMindDBResolver{db: db}, nil
}

func (r *MaxMindDBResolver) ResolveCountry(ipStr string) string {
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return i18n.T("common.na")
	}

	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}

	if err := r.db.Lookup(parsedIP, &record); err != nil {
		return i18n.T("common.na")
	}

	if record.Country.ISOCode == "" {
		return i18n.T("common.na")
	}

	return record.Country.ISOCode
}

func (r *MaxMindDBResolver) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

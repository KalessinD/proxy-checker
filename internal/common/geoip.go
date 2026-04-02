package common

import (
	"fmt"
	"net"
	"proxy-checker/internal/common/i18n"

	"github.com/oschwald/maxminddb-golang"
)

const DefaultLanguage = "en"

type (
	GeoIPResolver interface {
		ResolveCountry(ip, lang string) string
		Close() error
	}

	MaxMindDBResolver struct {
		DB *maxminddb.Reader
	}
)

func NewMaxMindDBResolverFromBytes(data []byte) (GeoIPResolver, error) {
	db, err := maxminddb.FromBytes(data)
	if err != nil {
		return nil, fmt.Errorf(i18n.T("geoip.err_open"), err)
	}
	return &MaxMindDBResolver{DB: db}, nil
}

func NewMaxMindDBResolverFromFile(filePath string) (GeoIPResolver, error) {
	db, err := maxminddb.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", i18n.T("geoip.err_open"), err)
	}
	return &MaxMindDBResolver{DB: db}, nil
}

func (r *MaxMindDBResolver) ResolveCountry(ipStr string, lang string) string {
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return i18n.T("common.na")
	}

	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
	}

	if err := r.DB.Lookup(parsedIP, &record); err != nil {
		return i18n.T("common.na")
	}

	if record.Country.ISOCode == "" {
		return i18n.T("common.na")
	}

	if name, ok := record.Country.Names[lang]; ok {
		return name
	}

	if name, ok := record.Country.Names[DefaultLanguage]; ok {
		return name
	}

	return record.Country.ISOCode
}

func (r *MaxMindDBResolver) Close() error {
	if r.DB != nil {
		return r.DB.Close()
	}
	return nil
}

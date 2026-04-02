package common

import _ "embed"

//go:embed "assets/geoip.mmdb"
var GeoIPData []byte

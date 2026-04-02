package common_test

import (
	"bytes"
	"net"
	"os"
	"proxy-checker/internal/common"
	"proxy-checker/internal/common/i18n"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	_ = i18n.Init("en")
	os.Exit(m.Run())
}

func createTestDB(t *testing.T) []byte {
	t.Helper()

	db, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType: "GeoIP2-Country",
			Languages:    []string{"en", "ru"},
		},
	)
	require.NoError(t, err)

	fullRecord := mmdbtype.Map{
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("US"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("United States"),
				"ru": mmdbtype.String("Соединенные Штаты"),
			},
		},
	}
	_, netIP, err := net.ParseCIDR("1.2.3.4/32")
	require.NoError(t, err)
	require.NoError(t, db.Insert(netIP, fullRecord))

	enOnlyRecord := mmdbtype.Map{
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("DE"),
			"names": mmdbtype.Map{
				"en": mmdbtype.String("Germany"),
				// "ru" отсутствует
			},
		},
	}
	_, netIP, err = net.ParseCIDR("5.6.7.8/32")
	require.NoError(t, err)
	require.NoError(t, db.Insert(netIP, enOnlyRecord))

	isoOnlyRecord := mmdbtype.Map{
		"country": mmdbtype.Map{
			"iso_code": mmdbtype.String("FR"),
		},
	}
	_, netIP, err = net.ParseCIDR("9.8.7.6/32")
	require.NoError(t, err)
	require.NoError(t, db.Insert(netIP, isoOnlyRecord))

	var buf bytes.Buffer
	_, err = db.WriteTo(&buf)
	require.NoError(t, err)

	return buf.Bytes()
}

func TestNewMaxMindDBResolverFromBytes_Success(t *testing.T) {
	testDBBytes := createTestDB(t)
	resolver, err := common.NewMaxMindDBResolverFromBytes(testDBBytes)
	require.NoError(t, err)
	require.NotNil(t, resolver)
	defer resolver.Close()
}

func TestNewMaxMindDBResolverFromBytes_InvalidData(t *testing.T) {
	invalidData := []byte("this is not a valid mmdb file")
	resolver, err := common.NewMaxMindDBResolverFromBytes(invalidData)
	require.Error(t, err)
	assert.Nil(t, resolver)
	assert.Contains(t, err.Error(), i18n.T("geoip.err_open"))
}

func TestNewMaxMindDBResolverFromFile_Success(t *testing.T) {
	testDBBytes := createTestDB(t)
	tempFile, err := os.CreateTemp(t.TempDir(), "test_geoip_*.mmdb")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())

	_, err = tempFile.Write(testDBBytes)
	require.NoError(t, err)
	tempFile.Close()

	resolver, err := common.NewMaxMindDBResolverFromFile(tempFile.Name())
	require.NoError(t, err)
	require.NotNil(t, resolver)
	defer resolver.Close()
}

func TestNewMaxMindDBResolverFromFile_NotFound(t *testing.T) {
	resolver, err := common.NewMaxMindDBResolverFromFile("/nonexistent/path/to/file.mmdb")
	require.Error(t, err)
	assert.Nil(t, resolver)
	assert.Contains(t, err.Error(), i18n.T("geoip.err_open"))
}

func TestMaxMindDBResolver_ResolveCountry(t *testing.T) {
	testDBBytes := createTestDB(t)
	resolver, err := common.NewMaxMindDBResolverFromBytes(testDBBytes)
	require.NoError(t, err)
	require.NotNil(t, resolver)
	defer resolver.Close()

	tests := []struct {
		name                string
		ipAddress           string
		language            string
		expectedCountryName string
	}{
		{
			name:                "Exact Russian match",
			ipAddress:           "1.2.3.4",
			language:            "ru",
			expectedCountryName: "Соединенные Штаты",
		},
		{
			name:                "Exact English match",
			ipAddress:           "1.2.3.4",
			language:            "en",
			expectedCountryName: "United States",
		},
		{
			name:                "Missing language fallback to English",
			ipAddress:           "5.6.7.8",
			language:            "ru", // В базе для DE нет русского
			expectedCountryName: "Germany",
		},
		{
			name:                "Missing language fallback to English (unknown lang)",
			ipAddress:           "5.6.7.8",
			language:            "es",
			expectedCountryName: "Germany",
		},
		{
			name:                "Missing names map fallback to ISO code",
			ipAddress:           "9.8.7.6",
			language:            "en",
			expectedCountryName: "FR",
		},
		{
			name:                "IP not found in database",
			ipAddress:           "192.168.1.1",
			language:            "en",
			expectedCountryName: i18n.T("common.na"),
		},
		{
			name:                "Invalid IP address format",
			ipAddress:           "not-an-ip",
			language:            "en",
			expectedCountryName: i18n.T("common.na"),
		},
		{
			name:                "Empty IP address string",
			ipAddress:           "",
			language:            "en",
			expectedCountryName: i18n.T("common.na"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ВАЖНО: Передаем язык вторым аргументом
			result := resolver.ResolveCountry(tt.ipAddress, tt.language)
			assert.Equal(t, tt.expectedCountryName, result)
		})
	}
}

func TestMaxMindDBResolver_Close_Idempotent(t *testing.T) {
	testDBBytes := createTestDB(t)
	resolver, err := common.NewMaxMindDBResolverFromBytes(testDBBytes)
	require.NoError(t, err)

	assert.NoError(t, resolver.Close())
	assert.NoError(t, resolver.Close())
}

func TestMaxMindDBResolver_Close_NilDB(t *testing.T) {
	resolver := &common.MaxMindDBResolver{DB: nil}
	assert.NoError(t, resolver.Close(), "Must be without panic")
}

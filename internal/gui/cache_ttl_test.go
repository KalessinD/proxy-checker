// nolint testpackage
package gui

import (
	"proxy-checker/internal/common"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatCacheTTLRemaining(t *testing.T) {
	testCases := []struct {
		name      string
		remaining time.Duration
		expected  string
	}{
		{name: "Zero duration", remaining: 0, expected: "0m"},
		{name: "Negative duration (expired)", remaining: -time.Minute, expected: "0m"},
		{name: "Less than a minute", remaining: 45 * time.Second, expected: "<1m"},
		{name: "Exact minute", remaining: 5 * time.Minute, expected: "5m"},
		{name: "Just under an hour", remaining: 59 * time.Minute, expected: "59m"},
		{name: "One hour and 5 minutes", remaining: 65 * time.Minute, expected: "1h 05m"},
		{name: "Two hours and zero minutes", remaining: 2 * time.Hour, expected: "2h 00m"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, formatCacheTTLRemaining(testCase.remaining))
		})
	}
}

func TestNearestCacheExpiry(t *testing.T) {
	t.Run("Empty map returns zero time", func(t *testing.T) {
		assert.True(t, nearestCacheExpiry(nil).IsZero(), "Nil map must produce a zero time")
		assert.True(t, nearestCacheExpiry(map[common.Source]time.Time{}).IsZero(), "Empty map must produce a zero time")
	})

	t.Run("Returns the earliest expiry", func(t *testing.T) {
		early := time.Now().Add(10 * time.Minute)
		late := time.Now().Add(2 * time.Hour)
		expiry := map[common.Source]time.Time{
			common.SourceProxyMania: late,
			common.SourceTheSpeedX:  early,
		}

		assert.Equal(t, early, nearestCacheExpiry(expiry), "The indicator must track the soonest expiring source")
	})
}

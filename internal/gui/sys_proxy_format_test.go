//go:build linux
// +build linux

package gui_test

import (
	"proxy-checker/internal/gui"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGVariantStringArray(t *testing.T) {
	tests := []struct {
		name         string
		rawInput     string
		expectedList []string
	}{
		{
			name:         "Empty array",
			rawInput:     "[]",
			expectedList: nil,
		},
		{
			name:         "Single element",
			rawInput:     "['localhost']",
			expectedList: []string{"localhost"},
		},
		{
			name:         "Multiple elements with spaces",
			rawInput:     "['localhost', '127.0.0.0/8', '::1']",
			expectedList: []string{"127.0.0.0/8", "::1", "localhost"},
		},
		{
			name:         "Elements without quotes (malformed but handled)",
			rawInput:     "[localhost, 192.168.1.1]",
			expectedList: []string{"192.168.1.1", "localhost"},
		},
		{
			name:         "Empty strings inside array",
			rawInput:     "['', 'test']",
			expectedList: []string{"test"},
		},
		{
			name:         "Trailing comma (GVariant sometimes does this)",
			rawInput:     "['localhost', ]",
			expectedList: []string{"localhost"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gui.ParseGVariantStringArray(tt.rawInput)
			assert.Equal(t, tt.expectedList, result)
		})
	}
}

func TestFormatGVariantStringArray(t *testing.T) {
	tests := []struct {
		name           string
		inputHosts     []string
		expectedString string
	}{
		{
			name:           "Nil slice returns empty array",
			inputHosts:     nil,
			expectedString: "[]",
		},
		{
			name:           "Empty slice returns empty array",
			inputHosts:     []string{},
			expectedString: "[]",
		},
		{
			name:           "Single element",
			inputHosts:     []string{"localhost"},
			expectedString: "['localhost']",
		},
		{
			name:           "Multiple elements are sorted and quoted",
			inputHosts:     []string{"192.168.1.1", "localhost", "::1"},
			expectedString: "['192.168.1.1', '::1', 'localhost']",
		},
		{
			name:           "Unsorted input gets sorted automatically",
			inputHosts:     []string{"zzz.com", "aaa.com"},
			expectedString: "['aaa.com', 'zzz.com']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gui.FormatGVariantStringArray(tt.inputHosts)
			assert.Equal(t, tt.expectedString, result)
		})
	}
}
